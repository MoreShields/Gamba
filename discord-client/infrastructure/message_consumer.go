package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	events "gambler/discord-client/proto/events"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// MessageConsumer manages NATS subscriptions and routes messages to handlers
type MessageConsumer struct {
	natsClient *NATSClient

	// Handler for LoL events
	lolHandler application.LoLEventHandler
	adapter    *ProtobufToLoLAdapter

	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMessageConsumer creates a new message consumer
func NewMessageConsumer(natsServers string, lolHandler application.LoLEventHandler) *MessageConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	// Create NATS client
	natsClient := NewNATSClient(natsServers)

	mc := &MessageConsumer{
		natsClient: natsClient,
		lolHandler: lolHandler,
		adapter:    NewProtobufToLoLAdapter(),
		ctx:        ctx,
		cancel:     cancel,
	}

	return mc
}

// Start begins consuming messages from all registered subjects
func (mc *MessageConsumer) Start(ctx context.Context) error {
	log.Info("Starting message consumer")

	// Connect to NATS
	if err := mc.natsClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Ensure required streams exist
	if err := mc.natsClient.EnsureLoLEventStream(); err != nil {
		return fmt.Errorf("failed to ensure LoL event stream: %w", err)
	}

	// Subscribe to LoL game state changes
	lolSubject := "lol.gamestate.*"
	if err := mc.subscribe(lolSubject); err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", lolSubject, err)
	}

	log.Info("Message consumer started and subscribed to LoL events")

	// Wait for shutdown signal
	<-mc.ctx.Done()

	// Clean up
	return mc.natsClient.Close()
}

// Stop gracefully shuts down the consumer
func (mc *MessageConsumer) Stop() {
	log.Info("Stopping message consumer")
	mc.cancel()
}

// subscribe sets up a subscription for a specific subject
func (mc *MessageConsumer) subscribe(subject string) error {
	return mc.natsClient.Subscribe(subject, func(data []byte) error {
		// Create a new context for this message
		ctx := context.Background()

		// Route based on subject pattern
		if strings.HasPrefix(subject, "lol.gamestate.") {
			return mc.handleLoLGameStateChange(ctx, data)
		}

		return fmt.Errorf("unhandled subject: %s", subject)
	})
}

// handleLoLGameStateChange processes LoL game state change events
func (mc *MessageConsumer) handleLoLGameStateChange(ctx context.Context, data []byte) error {
	// Deserialize the protobuf message
	event := &events.LoLGameStateChanged{}
	if err := proto.Unmarshal(data, event); err != nil {
		return fmt.Errorf("failed to unmarshal LoLGameStateChanged: %w", err)
	}

	log.WithFields(log.Fields{
		"summoner":       fmt.Sprintf("%s#%s", event.GameName, event.TagLine),
		"previousStatus": event.PreviousStatus,
		"currentStatus":  event.CurrentStatus,
		"gameId":         event.GameId,
	}).Debug("Processing LoL game state change")

	// Convert protobuf to domain DTO
	domainEvent, err := mc.adapter.ConvertGameStateChanged(event)
	if err != nil {
		// Log and ignore non-relevant transitions
		log.WithFields(log.Fields{
			"previousStatus": event.PreviousStatus,
			"currentStatus":  event.CurrentStatus,
		}).Debug("Ignoring non-relevant state transition")
		return nil
	}

	// Route to appropriate handler based on event type
	switch e := domainEvent.(type) {
	case dto.GameStartedDTO:
		return mc.lolHandler.HandleGameStarted(ctx, e)
	case dto.GameEndedDTO:
		return mc.lolHandler.HandleGameEnded(ctx, e)
	default:
		return fmt.Errorf("unexpected event type: %T", domainEvent)
	}
}
