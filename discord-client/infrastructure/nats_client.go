package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

// NATSClient implements the MessageBusClient interface using NATS with JetStream
type NATSClient struct {
	servers              string
	nc                   *nats.Conn
	js                   nats.JetStreamContext
	subscriptions        map[string]*nats.Subscription
	mu                   sync.RWMutex
	reconnectDelay       time.Duration
	maxReconnectAttempts int
}

// NewNATSClient creates a new NATS client
func NewNATSClient(servers string) *NATSClient {
	return &NATSClient{
		servers:              servers,
		subscriptions:        make(map[string]*nats.Subscription),
		reconnectDelay:       2 * time.Second,
		maxReconnectAttempts: 10,
	}
}

// Connect establishes a connection to the NATS server with JetStream
func (c *NATSClient) Connect(ctx context.Context) error {
	opts := []nats.Option{
		nats.Name("discord-client"),
		nats.MaxReconnects(c.maxReconnectAttempts),
		nats.ReconnectWait(c.reconnectDelay),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.WithError(err).Error("NATS disconnected with error")
			} else {
				log.Warn("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.WithFields(log.Fields{
				"subject": sub.Subject,
				"error":   err,
			}).Error("NATS async error")
		}),
	}

	// Connect to NATS
	nc, err := nats.Connect(c.servers, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	c.nc = nc
	c.js = js

	log.WithField("servers", c.servers).Info("Connected to NATS with JetStream")
	return nil
}

// Subscribe registers a handler for messages on the specified subject
// Uses JetStream for durable subscriptions
func (c *NATSClient) Subscribe(subject string, handler func([]byte) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.js == nil {
		return fmt.Errorf("not connected to NATS JetStream")
	}

	// Create a durable consumer name based on the subject
	// Replace invalid characters for consumer names
	sanitizedSubject := strings.ReplaceAll(subject, ".", "_")
	sanitizedSubject = strings.ReplaceAll(sanitizedSubject, "*", "wildcard")
	consumerName := fmt.Sprintf("discord-client-%s", sanitizedSubject)

	// Subscribe with manual acknowledgment
	sub, err := c.js.Subscribe(
		subject,
		func(msg *nats.Msg) {
			// Process the message
			if err := handler(msg.Data); err != nil {
				log.WithFields(log.Fields{
					"subject": subject,
					"error":   err,
				}).Error("Failed to process message")

				// Negative acknowledgment for retry
				if nakErr := msg.Nak(); nakErr != nil {
					log.WithError(nakErr).Error("Failed to NAK message")
				}
				return
			}

			// Acknowledge successful processing
			if ackErr := msg.Ack(); ackErr != nil {
				log.WithError(ackErr).Error("Failed to ACK message")
			}
		},
		nats.Durable(consumerName),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.MaxDeliver(3), // Retry up to 3 times
		nats.AckWait(30*time.Second),
	)

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	c.subscriptions[subject] = sub
	log.WithField("subject", subject).Info("Subscribed to NATS subject")
	return nil
}

// Close gracefully shuts down the NATS connection
func (c *NATSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unsubscribe from all subscriptions
	for subject, sub := range c.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.WithFields(log.Fields{
				"subject": subject,
				"error":   err,
			}).Error("Failed to unsubscribe")
		}
	}
	c.subscriptions = make(map[string]*nats.Subscription)

	// Close the connection
	if c.nc != nil {
		c.nc.Close()
		log.Info("NATS connection closed")
	}

	return nil
}

// IsConnected returns true if the client is connected to NATS
func (c *NATSClient) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}

// ensureStream ensures that the required JetStream stream exists
// This should be called during initialization to set up the lol_events stream
func (c *NATSClient) ensureStream(streamName string, subjects []string) error {
	if c.js == nil {
		return fmt.Errorf("not connected to NATS JetStream")
	}

	// Check if stream exists
	_, err := c.js.StreamInfo(streamName)
	if err == nil {
		log.WithField("stream", streamName).Info("JetStream stream already exists")
		return nil
	}

	// Create stream configuration
	cfg := &nats.StreamConfig{
		Name:        streamName,
		Subjects:    subjects,
		Retention:   nats.LimitsPolicy,
		MaxAge:      24 * time.Hour,
		MaxMsgs:     1000000,
		Storage:     nats.FileStorage,
		Replicas:    1,
		Description: "League of Legends game state events",
	}

	// Create the stream
	_, err = c.js.AddStream(cfg)
	if err != nil {
		return fmt.Errorf("failed to create stream %s: %w", streamName, err)
	}

	log.WithFields(log.Fields{
		"stream":   streamName,
		"subjects": subjects,
	}).Info("Created JetStream stream")
	return nil
}

// EnsureLoLEventStream ensures the lol_events stream exists
// This should be called after connection is established
func (c *NATSClient) EnsureLoLEventStream() error {
	return c.ensureStream("lol_events", []string{"lol.gamestate.*"})
}

// Publish publishes a message to the specified subject using JetStream
func (c *NATSClient) Publish(ctx context.Context, subject string, data []byte) error {
	if c.js == nil {
		return fmt.Errorf("not connected to NATS JetStream")
	}

	_, err := c.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", subject, err)
	}

	log.WithFields(log.Fields{
		"subject": subject,
		"size":    len(data),
	}).Debug("Published message to NATS")
	return nil
}
