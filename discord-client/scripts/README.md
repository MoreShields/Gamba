# Scripts

This directory contains utility scripts for the Discord client.

## fetch_message.go

Fetches a Discord message by ID and outputs it in various formats for testing purposes.

### Usage

```bash
go run fetch_message.go -token "YOUR_BOT_TOKEN" -channel "CHANNEL_ID" -message "MESSAGE_ID" -output "messageCreate"
```

### Options

- `-token`: Discord bot token (can also use DISCORD_TOKEN env var)
- `-channel`: Channel ID where the message is located
- `-message`: Message ID to fetch
- `-output`: Output format (default: "json")
  - `json`: Raw JSON of the message
  - `go`: Go struct literal
  - `messageCreate`: Reconstructed MessageCreate event for testing

### Example

```bash
# Set token as environment variable
export DISCORD_TOKEN="your_bot_token"

# Fetch message and output as MessageCreate
go run fetch_message.go -channel "123456789" -message "987654321" -output "messageCreate"
```

## LoL Event Testing Scripts

This directory contains development tools for testing the LoL game state event system.

## publish_lol_events.go

A CLI tool for publishing simulated LoL game state change events to NATS for rapid development testing.

### Prerequisites

1. **NATS Server Running**: Ensure NATS with JetStream is running
   ```bash
   # Using Docker
   docker run -p 4222:4222 nats:latest -js
   
   # Or using nats-server binary
   nats-server -js
   ```

2. **Discord Bot Running**: Start the discord-client to consume events
   ```bash
   # Set required environment variables
   export DISCORD_TOKEN="your_bot_token"
   export DATABASE_URL="your_db_url" 
   export NATS_SERVERS="nats://localhost:4222"
   
   # Run the bot
   go run main.go
   ```

### Usage Examples

#### Basic Game Events

```bash
# Start a ranked game for Faker
go run scripts/publish_lol_events.go --event=start --summoner=Feviben --tag=NA1 --queue=RANKED_SOLO_5x5

# End the game with a win (Azir, 30 minutes)
go run scripts/publish_lol_events.go --event=end --summoner=Feviben --tag=NA1 --win=true --champion=Azir --duration=1800

# Complete game cycle with 10 second delay
go run scripts/publish_lol_events.go --event=cycle --summoner=Doublelift --tag=NA1 --delay=10s
```

#### Batch Testing

```bash
# Test multiple summoners simultaneously
go run scripts/publish_lol_events.go --event=batch --summoners='Faker#KR1,Caps#EUW1,Doublelift#NA1,Bjergsen#NA1'
```

#### Development Options

```bash
# Dry run (print events without publishing)
go run scripts/publish_lol_events.go --event=start --summoner=TestPlayer --tag=NA1 --dry-run

# Verbose logging
go run scripts/publish_lol_events.go --event=cycle --summoner=TestPlayer --tag=NA1 --verbose

# Custom NATS server
go run scripts/publish_lol_events.go --event=start --summoner=TestPlayer --tag=NA1 --nats=nats://your-server:4222
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--event` | `start` | Event type: `start`, `end`, `cycle`, `batch` |
| `--summoner` | `Faker` | Summoner name |
| `--tag` | `KR1` | Riot ID tag line |
| `--queue` | `RANKED_SOLO_5x5` | Queue type |
| `--game-id` | auto-generated | Game ID |
| `--champion` | `Azir` | Champion played (for end events) |
| `--win` | `true` | Did the player win |
| `--duration` | `1800` | Game duration in seconds |
| `--delay` | `30s` | Delay between start/end for cycle events |
| `--summoners` | - | Comma-separated list for batch mode |
| `--nats` | `nats://localhost:4222` | NATS server address |
| `--dry-run` | `false` | Print messages without publishing |
| `--verbose` | `false` | Enable verbose logging |

### Event Types

- **start**: Publishes a game start event (`NOT_IN_GAME` → `IN_GAME`)
- **end**: Publishes a game end event (`IN_GAME` → `NOT_IN_GAME` with results)
- **cycle**: Publishes start, waits, then publishes end
- **batch**: Publishes cycles for multiple summoners

### Testing Workflow

1. **Start Discord Bot**: Ensure the bot is running and watching for summoners
2. **Add Summoner Watch**: Use Discord `/summoner add` command to watch test summoners
3. **Publish Events**: Use this script to trigger game events
4. **Verify Behavior**: Check Discord for house wager creation and user interactions

### Troubleshooting

- **"Failed to connect to NATS"**: Ensure NATS server is running with JetStream enabled
- **"No guilds watching this summoner"**: Add the summoner via Discord `/summoner add` command first
- **Events not reaching Discord**: Check bot logs for processing errors
- **Permission errors**: Ensure bot has message posting permissions in target channels

### Example Testing Session

```bash
# 1. Start a test game
go run scripts/publish_lol_events.go --event=start --summoner=Feviben --tag=NA1 --verbose

# 2. Check Discord for house wager creation
# 3. Place some test bets via Discord UI
# 4. End the game with a win

go run scripts/publish_lol_events.go --event=end --summoner=Feviben --tag=NA1 --win=true --champion=Yasuo --verbose

# 5. Verify wager resolution in Discord
```

This creates a complete testing loop to validate the entire LoL event → Discord integration without waiting for real games.