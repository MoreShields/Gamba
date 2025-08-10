# OpenTelemetry Instrumentation for Discord Client

This package provides OpenTelemetry metrics instrumentation for the Discord client service, following a similar pattern to the lol-tracker Python service.

## Overview

The observability package implements comprehensive metrics collection for:
- Discord message processing
- Active wager tracking
- NATS message flow
- User balance transactions
- Database query performance

## Configuration

The OpenTelemetry instrumentation is configured through environment variables:

- `OTEL_ENABLED` - Enable/disable metrics collection (default: true)
- `OTEL_EXPORTER_TYPE` - Exporter type: "otlp", "console", or "none" (default: otlp)
- `OTEL_OTLP_ENDPOINT` - OTLP collector endpoint (default: localhost:4317)
- `OTEL_SERVICE_NAME` - Service name for metrics (default: discord-client)
- `OTEL_EXPORT_INTERVAL_MILLIS` - Export interval in milliseconds (default: 5000)
- `OTEL_EXPORT_TIMEOUT_MILLIS` - Export timeout in milliseconds (default: 10000)

## Metrics

### Discord Messages
- `discord_client.messages.read_total` - Count of Discord messages read by type (command, interaction, message)

### Wagers
- `discord_client.wagers.active` - Gauge showing current number of active wagers by type (lol, tft, group)

### NATS Messages
- `discord_client.nats.messages_received_total` - Count of messages received by event type
- `discord_client.nats.messages_published_total` - Count of messages published by event type

### User Balances
- `discord_client.balance.transactions_total` - Count of balance transactions by type (wager, bet, transfer, wordle_reward, daily_award, high_roller, refund)

### Database Operations
- `discord_client.database.queries_total` - Count of database queries by repository and method
- `discord_client.database.query_duration` - Histogram of query duration in seconds by repository and method

## Usage

The metrics provider is initialized automatically when the service starts if `OTEL_ENABLED=true`.

### Recording Metrics

```go
import "gambler/discord-client/infrastructure/observability"

// Get the global metrics provider
metrics := observability.GetMetrics()

// Record a Discord message
metrics.RecordMessageRead(observability.MessageTypeCommand)

// Update active wagers count
metrics.UpdateActiveWagers(observability.WagerTypeLoL, 1)  // increment
metrics.UpdateActiveWagers(observability.WagerTypeLoL, -1) // decrement

// Record NATS messages
metrics.RecordNATSMessageReceived("lol_game_started")
metrics.RecordNATSMessagePublished("wager_created")

// Record balance transaction
metrics.RecordBalanceTransaction(observability.TransactionTypeWager)

// Record database query with duration
defer metrics.MeasureDatabaseQuery("user", "GetByDiscordID")()
```

## Implementation Status

### Completed
- ✅ Core metrics provider infrastructure
- ✅ OpenTelemetry configuration integration
- ✅ Metrics initialization in service startup
- ✅ Graceful shutdown handling
- ✅ Constants and metric definitions

### Pending
- ⏳ Repository layer instrumentation
- ⏳ NATS message consumer/publisher instrumentation
- ⏳ Discord bot command/interaction instrumentation
- ⏳ Active wager gauge updates in domain services
- ⏳ Balance transaction tracking

## Testing

To test with console output:
```bash
OTEL_ENABLED=true OTEL_EXPORTER_TYPE=console ./discord-client
```

To send to an OTLP collector:
```bash
OTEL_ENABLED=true OTEL_EXPORTER_TYPE=otlp OTEL_OTLP_ENDPOINT=localhost:4317 ./discord-client
```

## Next Steps

1. Add instrumentation to repositories by wrapping database operations
2. Instrument NATS message handling in `infrastructure/message_consumer.go`
3. Add Discord message tracking in `bot/bot.go`
4. Track wager state changes in domain services
5. Monitor balance transactions in the balance service