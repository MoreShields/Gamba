package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"gambler/discord-client/application"
	log "github.com/sirupsen/logrus"
)

// MessageHandler defines a function that handles raw message bytes
type MessageHandler func(ctx context.Context, data []byte) error

// MessageConsumer manages NATS subscriptions and routes messages to handlers
type MessageConsumer struct {
	natsClient  *NATSClient
	lolListener *LoLEventListener
	handlers    map[string]MessageHandler
	mu          sync.RWMutex
	
	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMessageConsumer creates a new message consumer with all handlers configured
func NewMessageConsumer(natsServers string, lolHandler application.LoLHandler) *MessageConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create NATS client
	natsClient := NewNATSClient(natsServers)
	
	// Create LoL event listener
	lolListener := NewLoLEventListener(lolHandler)
	
	mc := &MessageConsumer{
		natsClient:  natsClient,
		lolListener: lolListener,
		handlers:    make(map[string]MessageHandler),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Register LoL event handler
	mc.RegisterHandler("lol.gamestate.*", lolListener.HandleLoLGameStateChange)
	
	return mc
}

// RegisterHandler registers a handler for a specific subject pattern
func (mc *MessageConsumer) RegisterHandler(subject string, handler MessageHandler) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.handlers[subject] = handler
	log.WithField("subject", subject).Info("Registered message handler")
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
	
	// Subscribe to all registered subjects
	mc.mu.RLock()
	subjects := make([]string, 0, len(mc.handlers))
	for subject := range mc.handlers {
		subjects = append(subjects, subject)
	}
	mc.mu.RUnlock()
	
	// Subscribe to each subject
	for _, subject := range subjects {
		if err := mc.subscribe(subject); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}
	}
	
	log.WithField("subjects", subjects).Info("Message consumer started and subscribed to subjects")
	
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
		mc.mu.RLock()
		handler, exists := mc.handlers[subject]
		mc.mu.RUnlock()
		
		if !exists {
			return fmt.Errorf("no handler registered for subject: %s", subject)
		}
		
		// Create a new context for this message
		ctx := context.Background()
		
		// Handle the message
		if err := handler(ctx, data); err != nil {
			log.WithFields(log.Fields{
				"subject": subject,
				"error":   err,
			}).Error("Failed to handle message")
			return err
		}
		
		return nil
	})
}