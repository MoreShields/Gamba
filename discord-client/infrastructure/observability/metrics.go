package observability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gambler/discord-client/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// MetricsProvider manages OpenTelemetry metrics for the discord-client service
type MetricsProvider struct {
	config         *config.Config
	meterProvider  *sdkmetric.MeterProvider
	meter          metric.Meter
	initialized    bool
	mu             sync.RWMutex

	// Metric instruments
	messagesReadCounter         metric.Int64Counter
	wagersActiveGauge          metric.Int64UpDownCounter
	natsMessagesReceivedCounter  metric.Int64Counter
	natsMessagesPublishedCounter metric.Int64Counter
	balanceTransactionsCounter   metric.Int64Counter
	databaseQueriesCounter       metric.Int64Counter
	databaseQueryDurationHist    metric.Float64Histogram
}

// NewMetricsProvider creates a new metrics provider
func NewMetricsProvider(cfg *config.Config) *MetricsProvider {
	return &MetricsProvider{
		config: cfg,
	}
}

// Initialize sets up the OpenTelemetry metrics provider
func (mp *MetricsProvider) Initialize(ctx context.Context) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.initialized {
		log.Println("Metrics provider already initialized")
		return nil
	}

	if !mp.config.OTelEnabled {
		log.Println("OpenTelemetry metrics disabled")
		mp.initialized = true
		return nil
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(mp.config.OTelServiceName),
			attribute.String("environment", mp.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create appropriate exporter based on config
	var exporter sdkmetric.Exporter
	switch mp.config.OTelExporterType {
	case "console":
		exporter, err = stdoutmetric.New()
		if err != nil {
			return fmt.Errorf("failed to create console exporter: %w", err)
		}
		log.Println("Using console metric exporter")

	case "otlp":
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(mp.config.OTelOTLPEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		log.Printf("Using OTLP metric exporter: %s", mp.config.OTelOTLPEndpoint)

	case "none":
		log.Println("Metrics export disabled (exporter_type='none')")
		mp.initialized = true
		return nil

	default:
		return fmt.Errorf("unknown exporter type: %s", mp.config.OTelExporterType)
	}

	// Create meter provider with periodic reader
	mp.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(time.Duration(mp.config.OTelExportIntervalMillis)*time.Millisecond),
			),
		),
	)

	// Set as global meter provider
	otel.SetMeterProvider(mp.meterProvider)

	// Get meter for creating instruments
	mp.meter = mp.meterProvider.Meter("discord-client")

	// Create metric instruments
	if err := mp.createInstruments(); err != nil {
		return fmt.Errorf("failed to create instruments: %w", err)
	}

	mp.initialized = true
	log.Println("Metrics provider initialized successfully")
	return nil
}

// createInstruments creates all metric instruments
func (mp *MetricsProvider) createInstruments() error {
	var err error

	// Discord metrics
	mp.messagesReadCounter, err = mp.meter.Int64Counter(
		MessagesReadTotal,
		metric.WithDescription("Total number of Discord messages read"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create messages read counter: %w", err)
	}

	// Wager metrics - using UpDownCounter for gauge-like behavior
	mp.wagersActiveGauge, err = mp.meter.Int64UpDownCounter(
		WagersActive,
		metric.WithDescription("Current number of active wagers"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create wagers active gauge: %w", err)
	}

	// NATS metrics
	mp.natsMessagesReceivedCounter, err = mp.meter.Int64Counter(
		NATSMessagesReceivedTotal,
		metric.WithDescription("Total number of NATS messages received"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create NATS messages received counter: %w", err)
	}

	mp.natsMessagesPublishedCounter, err = mp.meter.Int64Counter(
		NATSMessagesPublishedTotal,
		metric.WithDescription("Total number of NATS messages published"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create NATS messages published counter: %w", err)
	}

	// Balance metrics
	mp.balanceTransactionsCounter, err = mp.meter.Int64Counter(
		BalanceTransactionsTotal,
		metric.WithDescription("Total number of balance transactions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create balance transactions counter: %w", err)
	}

	// Database metrics
	mp.databaseQueriesCounter, err = mp.meter.Int64Counter(
		DatabaseQueriesTotal,
		metric.WithDescription("Total number of database queries"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create database queries counter: %w", err)
	}

	mp.databaseQueryDurationHist, err = mp.meter.Float64Histogram(
		DatabaseQueryDuration,
		metric.WithDescription("Duration of database queries in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create database query duration histogram: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the metrics provider
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.meterProvider != nil {
		return mp.meterProvider.Shutdown(ctx)
	}
	return nil
}

// RecordMessageRead records a Discord message being read
func (mp *MetricsProvider) RecordMessageRead(messageType string) {
	if !mp.isEnabled() {
		return
	}

	mp.messagesReadCounter.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String(LabelType, messageType),
		),
	)
}

// UpdateActiveWagers updates the count of active wagers (increment/decrement)
func (mp *MetricsProvider) UpdateActiveWagers(wagerType string, delta int64) {
	if !mp.isEnabled() {
		return
	}

	mp.wagersActiveGauge.Add(context.Background(), delta,
		metric.WithAttributes(
			attribute.String(LabelType, wagerType),
		),
	)
}

// RecordNATSMessageReceived records a NATS message being received
func (mp *MetricsProvider) RecordNATSMessageReceived(eventType string) {
	if !mp.isEnabled() {
		return
	}

	mp.natsMessagesReceivedCounter.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String(LabelEventType, eventType),
		),
	)
}

// RecordNATSMessagePublished records a NATS message being published
func (mp *MetricsProvider) RecordNATSMessagePublished(eventType string) {
	if !mp.isEnabled() {
		return
	}

	mp.natsMessagesPublishedCounter.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String(LabelEventType, eventType),
		),
	)
}

// RecordBalanceTransaction records a balance transaction
func (mp *MetricsProvider) RecordBalanceTransaction(transactionType string) {
	if !mp.isEnabled() {
		return
	}

	mp.balanceTransactionsCounter.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String(LabelType, transactionType),
		),
	)
}

// RecordDatabaseQuery records a database query with duration
func (mp *MetricsProvider) RecordDatabaseQuery(repository, method string, duration time.Duration) {
	if !mp.isEnabled() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String(LabelRepository, repository),
		attribute.String(LabelMethod, method),
	)

	mp.databaseQueriesCounter.Add(context.Background(), 1, attrs)
	mp.databaseQueryDurationHist.Record(context.Background(), duration.Seconds(), attrs)
}

// MeasureDatabaseQuery returns a function to measure database query duration
// Usage:
//
//	defer mp.MeasureDatabaseQuery("user", "GetByDiscordID")()
func (mp *MetricsProvider) MeasureDatabaseQuery(repository, method string) func() {
	start := time.Now()
	return func() {
		mp.RecordDatabaseQuery(repository, method, time.Since(start))
	}
}

// isEnabled checks if metrics are enabled and initialized
func (mp *MetricsProvider) isEnabled() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.initialized && mp.config.OTelEnabled
}

// Global metrics provider instance
var (
	globalMetrics *MetricsProvider
	metricsOnce   sync.Once
)

// InitializeGlobalMetrics initializes the global metrics provider
func InitializeGlobalMetrics(ctx context.Context, cfg *config.Config) error {
	var err error
	metricsOnce.Do(func() {
		globalMetrics = NewMetricsProvider(cfg)
		err = globalMetrics.Initialize(ctx)
	})
	return err
}

// GetMetrics returns the global metrics provider
func GetMetrics() *MetricsProvider {
	return globalMetrics
}

// ShutdownGlobalMetrics shuts down the global metrics provider
func ShutdownGlobalMetrics(ctx context.Context) error {
	if globalMetrics != nil {
		return globalMetrics.Shutdown(ctx)
	}
	return nil
}