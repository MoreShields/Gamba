package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gambler/discord-client/infrastructure"
	events "gambler/discord-client/proto/events"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventType represents the type of event to publish
type EventType string

const (
	EventStart EventType = "start"
	EventEnd   EventType = "end"
	EventCycle EventType = "cycle"
	EventBatch EventType = "batch"
)

// Config holds script configuration
type Config struct {
	Event        EventType
	SummonerName string
	TagLine      string
	QueueType    string
	GameID       string
	Champion     string
	Win          bool
	Duration     int32
	Delay        time.Duration
	Summoners    string
	NATSServers  string
	DryRun       bool
	Verbose      bool
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.Printf("Starting LoL event publisher with config: %+v", config)
	}

	ctx := context.Background()
	publisher := NewEventPublisher(config.NATSServers, config.DryRun, config.Verbose)

	if err := publisher.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer publisher.Close()

	switch config.Event {
	case EventStart:
		if err := publisher.PublishGameStart(ctx, config); err != nil {
			log.Fatalf("Failed to publish game start: %v", err)
		}
	case EventEnd:
		if err := publisher.PublishGameEnd(ctx, config); err != nil {
			log.Fatalf("Failed to publish game end: %v", err)
		}
	case EventCycle:
		if err := publisher.PublishGameCycle(ctx, config); err != nil {
			log.Fatalf("Failed to publish game cycle: %v", err)
		}
	case EventBatch:
		if err := publisher.PublishBatch(ctx, config); err != nil {
			log.Fatalf("Failed to publish batch: %v", err)
		}
	default:
		log.Fatalf("Unknown event type: %s", config.Event)
	}

	log.Println("Event publishing completed successfully")
}

// parseFlags parses command-line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}

	var eventStr string
	flag.StringVar(&eventStr, "event", "start", "Event type: start, end, cycle, batch")
	flag.StringVar(&config.SummonerName, "summoner", "Faker", "Summoner name")
	flag.StringVar(&config.TagLine, "tag", "KR1", "Riot ID tag line")
	flag.StringVar(&config.QueueType, "queue", "RANKED_SOLO_5x5", "Queue type")
	flag.StringVar(&config.GameID, "game-id", "", "Game ID (auto-generated if empty)")
	flag.StringVar(&config.Champion, "champion", "Azir", "Champion played")
	flag.BoolVar(&config.Win, "win", true, "Did the player win")
	var duration int
	flag.IntVar(&duration, "duration", 1800, "Game duration in seconds")
	flag.DurationVar(&config.Delay, "delay", 30*time.Second, "Delay between start and end for cycle events")
	flag.StringVar(&config.Summoners, "summoners", "", "Comma-separated list of summoner#tag pairs for batch mode")
	flag.StringVar(&config.NATSServers, "nats", "nats://localhost:4222", "NATS server addresses")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Print messages without publishing")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Publish LoL game state change events to NATS for testing.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Start a ranked game for Faker\n")
		fmt.Fprintf(os.Stderr, "  %s --event=start --summoner=Faker --tag=KR1 --queue=RANKED_SOLO_5x5\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # End a game with a win\n")
		fmt.Fprintf(os.Stderr, "  %s --event=end --summoner=Faker --tag=KR1 --win=true --champion=Azir\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Full game cycle\n")
		fmt.Fprintf(os.Stderr, "  %s --event=cycle --summoner=Doublelift --tag=NA1 --delay=30s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Batch test multiple summoners\n")
		fmt.Fprintf(os.Stderr, "  %s --event=batch --summoners='Faker#KR1,Caps#EUW1,Doublelift#NA1'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Convert duration to int32
	config.Duration = int32(duration)

	// Parse event type
	switch strings.ToLower(eventStr) {
	case "start":
		config.Event = EventStart
	case "end":
		config.Event = EventEnd
	case "cycle":
		config.Event = EventCycle
	case "batch":
		config.Event = EventBatch
	default:
		log.Fatalf("Invalid event type: %s. Must be one of: start, end, cycle, batch", eventStr)
	}

	// Generate game ID if not provided
	if config.GameID == "" {
		config.GameID = fmt.Sprintf("test-game-%d", time.Now().Unix())
	}

	// Validate batch mode
	if config.Event == EventBatch && config.Summoners == "" {
		log.Fatalf("Batch mode requires --summoners parameter")
	}

	return config
}

// EventPublisher handles publishing events to NATS
type EventPublisher struct {
	natsServers string
	natsClient  *infrastructure.NATSClient
	dryRun      bool
	verbose     bool
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(natsServers string, dryRun, verbose bool) *EventPublisher {
	return &EventPublisher{
		natsServers: natsServers,
		natsClient:  infrastructure.NewNATSClient(natsServers),
		dryRun:      dryRun,
		verbose:     verbose,
	}
}

// Connect establishes connection to NATS
func (p *EventPublisher) Connect(ctx context.Context) error {
	if p.dryRun {
		log.Println("DRY RUN: Skipping NATS connection")
		return nil
	}

	if err := p.natsClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Ensure LoL event stream exists
	if err := p.natsClient.EnsureLoLEventStream(); err != nil {
		return fmt.Errorf("failed to ensure LoL event stream: %w", err)
	}

	if p.verbose {
		log.Println("Connected to NATS and ensured LoL event stream")
	}

	return nil
}

// Close closes the NATS connection
func (p *EventPublisher) Close() {
	if !p.dryRun && p.natsClient != nil {
		p.natsClient.Close()
	}
}

// PublishGameStart publishes a game start event
func (p *EventPublisher) PublishGameStart(ctx context.Context, config *Config) error {
	event := &events.LoLGameStateChanged{
		GameName:       config.SummonerName,
		TagLine:        config.TagLine,
		PreviousStatus: events.GameStatus_GAME_STATUS_NOT_IN_GAME,
		CurrentStatus:  events.GameStatus_GAME_STATUS_IN_GAME,
		EventTime:      timestamppb.Now(),
		GameId:         &config.GameID,
		QueueType:      &config.QueueType,
	}

	return p.publishEvent(ctx, event, "Game Started")
}

// PublishGameEnd publishes a game end event
func (p *EventPublisher) PublishGameEnd(ctx context.Context, config *Config) error {
	gameResult := &events.GameResult{
		Won:             config.Win,
		DurationSeconds: config.Duration,
		QueueType:       config.QueueType,
		ChampionPlayed:  config.Champion,
	}

	event := &events.LoLGameStateChanged{
		GameName:       config.SummonerName,
		TagLine:        config.TagLine,
		PreviousStatus: events.GameStatus_GAME_STATUS_IN_GAME,
		CurrentStatus:  events.GameStatus_GAME_STATUS_NOT_IN_GAME,
		GameResult:     gameResult,
		EventTime:      timestamppb.Now(),
		GameId:         &config.GameID,
		QueueType:      &config.QueueType,
	}

	return p.publishEvent(ctx, event, "Game Ended")
}

// PublishGameCycle publishes a complete game cycle (start -> wait -> end)
func (p *EventPublisher) PublishGameCycle(ctx context.Context, config *Config) error {
	log.Printf("Publishing game cycle for %s#%s (delay: %v)", config.SummonerName, config.TagLine, config.Delay)

	// Publish game start
	if err := p.PublishGameStart(ctx, config); err != nil {
		return fmt.Errorf("failed to publish game start: %w", err)
	}

	log.Printf("Waiting %v before publishing game end...", config.Delay)
	time.Sleep(config.Delay)

	// Publish game end
	if err := p.PublishGameEnd(ctx, config); err != nil {
		return fmt.Errorf("failed to publish game end: %w", err)
	}

	return nil
}

// PublishBatch publishes events for multiple summoners
func (p *EventPublisher) PublishBatch(ctx context.Context, config *Config) error {
	summoners := strings.Split(config.Summoners, ",")
	log.Printf("Publishing batch events for %d summoners", len(summoners))

	for i, summoner := range summoners {
		parts := strings.Split(strings.TrimSpace(summoner), "#")
		if len(parts) != 2 {
			log.Printf("Skipping invalid summoner format: %s (expected name#tag)", summoner)
			continue
		}

		summonerConfig := *config // Copy config
		summonerConfig.SummonerName = parts[0]
		summonerConfig.TagLine = parts[1]
		summonerConfig.GameID = fmt.Sprintf("batch-game-%d-%d", time.Now().Unix(), i)

		// Vary some parameters for interesting test data
		summonerConfig.Win = i%2 == 0 // Alternate win/loss
		champions := []string{"Azir", "Faker", "Yasuo", "Zed", "LeBlanc", "Syndra"}
		summonerConfig.Champion = champions[i%len(champions)]
		summonerConfig.Duration = 1500 + int32(i*300) // Vary game duration

		log.Printf("Publishing cycle for %s#%s", summonerConfig.SummonerName, summonerConfig.TagLine)
		if err := p.PublishGameCycle(ctx, &summonerConfig); err != nil {
			log.Printf("Failed to publish cycle for %s#%s: %v", summonerConfig.SummonerName, summonerConfig.TagLine, err)
			continue
		}

		// Small delay between summoners
		if i < len(summoners)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// publishEvent serializes and publishes an event to NATS
func (p *EventPublisher) publishEvent(ctx context.Context, event *events.LoLGameStateChanged, eventType string) error {
	// Serialize protobuf message
	data, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	subject := fmt.Sprintf("lol.gamestate.changed")

	if p.verbose || p.dryRun {
		log.Printf("Event: %s", eventType)
		log.Printf("Subject: %s", subject)
		log.Printf("Summoner: %s#%s", event.GameName, event.TagLine)
		log.Printf("Transition: %s -> %s", event.PreviousStatus, event.CurrentStatus)
		if event.GameId != nil {
			log.Printf("Game ID: %s", *event.GameId)
		}
		if event.QueueType != nil {
			log.Printf("Queue: %s", *event.QueueType)
		}
		if event.GameResult != nil {
			result := "Loss"
			if event.GameResult.Won {
				result = "Win"
			}
			log.Printf("Result: %s (%s, %ds)", result, event.GameResult.ChampionPlayed, event.GameResult.DurationSeconds)
		}
		log.Printf("Message size: %d bytes", len(data))
		log.Println("---")
	}

	if p.dryRun {
		log.Printf("DRY RUN: Would publish %s event for %s#%s", eventType, event.GameName, event.TagLine)
		return nil
	}

	// Publish to NATS
	if err := p.natsClient.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	log.Printf("Published %s event for %s#%s", eventType, event.GameName, event.TagLine)
	return nil
}
