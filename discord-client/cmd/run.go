package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gambler/discord-client/application"
	"gambler/discord-client/bot"
	"gambler/discord-client/bot/features/betting"
	"gambler/discord-client/config"
	"gambler/discord-client/database"
	"gambler/discord-client/infrastructure"

	summoner_pb "gambler/discord-client/proto/services"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Run initializes and starts the application
func Run(ctx context.Context) error {
	// Configure logrus
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // default to info level
	}
	
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Printf("Invalid LOG_LEVEL '%s', defaulting to info: %v", logLevel, err)
		level = logrus.InfoLevel
	}
	
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	
	log.Printf("Starting gambler bot with log level: %s", level)

	// Load configuration
	cfg := config.Get()

	/// Initialize infrastructure connections
	db, err := initializeDatabase(ctx, cfg)
	if err != nil {
		return err
	}

	natsClient, err := initializeNATS(ctx, cfg)
	if err != nil {
		return err
	}

	summonerConn, summonerClient, err := initializeSummonerClient(cfg)
	if err != nil {
		return err
	}

	/// Initialize event infrastructure
	subjectMapper, natsEventPublisher, err := initializeEventInfrastructure(natsClient)
	if err != nil {
		return err
	}

	/// Initialize repositories and services
	uowFactory := initializeRepositories(db, natsEventPublisher)

	// Initialize Discord bot
	discordBot, err := initializeDiscordBot(cfg, uowFactory, summonerClient, natsEventPublisher)
	if err != nil {
		return err
	}

	// Initialize application handlers
	lolHandler := initializeApplicationHandlers(uowFactory, discordBot)

	// Initialize application workers
	dailyAwardsWorker := initializeApplicationWorkers(uowFactory, discordBot)

	// Setup event subscriptions
	if err := setupEventSubscriptions(natsClient, subjectMapper, uowFactory, discordBot, cfg); err != nil {
		return err
	}

	// Start background services
	messageConsumer, cleanupFuncs := startBackgroundServices(ctx, cfg, lolHandler, dailyAwardsWorker, discordBot)

	// Wait for shutdown signal
	log.Printf("Bot is running in %s mode...", cfg.Environment)
	<-ctx.Done()

	// Graceful shutdown
	performGracefulShutdown(messageConsumer, discordBot, natsClient, summonerConn, db, cleanupFuncs)

	return nil
}

// creates and returns a database connection
func initializeDatabase(ctx context.Context, cfg *config.Config) (*database.DB, error) {
	log.Println("Connecting to database...")
	db, err := database.NewConnection(ctx, cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Database connection established successfully")
	return db, nil
}

// creates and connects to NATS
func initializeNATS(ctx context.Context, cfg *config.Config) (*infrastructure.NATSClient, error) {
	log.Printf("Initializing NATS client with servers: %s...", cfg.NATSServers)
	natsClient := infrastructure.NewNATSClient(cfg.NATSServers)
	if err := natsClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	log.Println("NATS client connected successfully")
	return natsClient, nil
}

// creates gRPC connection to summoner tracking service
func initializeSummonerClient(cfg *config.Config) (*grpc.ClientConn, summoner_pb.SummonerTrackingServiceClient, error) {
	log.Printf("Connecting to summoner tracking service at %s...", cfg.SummonerServiceAddr)
	summonerConn, err := grpc.NewClient(cfg.SummonerServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to summoner tracking service: %w", err)
	}
	summonerClient := summoner_pb.NewSummonerTrackingServiceClient(summonerConn)
	log.Println("Summoner tracking service connection established successfully")
	return summonerConn, summonerClient, nil
}

// sets up event publishing infrastructure
func initializeEventInfrastructure(natsClient *infrastructure.NATSClient) (*infrastructure.EventSubjectMapper, *infrastructure.NATSEventPublisher, error) {
	log.Println("Initializing event infrastructure...")
	subjectMapper := infrastructure.NewEventSubjectMapper()
	natsEventPublisher := infrastructure.NewNATSEventPublisher(natsClient, subjectMapper)

	// Ensure domain events stream exists
	if err := natsEventPublisher.EnsureDomainEventStream(); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure domain events stream: %w", err)
	}
	log.Println("Event infrastructure initialized successfully")
	return subjectMapper, natsEventPublisher, nil
}

// creates the unit of work factory
func initializeRepositories(db *database.DB, eventPublisher *infrastructure.NATSEventPublisher) *infrastructure.UnitOfWorkFactory {
	log.Println("Initializing unit of work factory...")
	uowFactory := infrastructure.NewUnitOfWorkFactory(db, eventPublisher)
	log.Println("Unit of work factory initialized successfully")
	return uowFactory
}

// creates and configures the Discord bot
func initializeDiscordBot(cfg *config.Config, uowFactory application.UnitOfWorkFactory, summonerClient summoner_pb.SummonerTrackingServiceClient, eventPublisher *infrastructure.NATSEventPublisher) (*bot.Bot, error) {
	log.Println("Initializing Discord bot...")
	botConfig := bot.Config{
		Token:          cfg.DiscordToken,
		GuildID:        cfg.GuildID,
		GambaChannelID: cfg.GambaChannelID,
	}
	gamblingConfig := &betting.GamblingConfig{
		DailyGambleLimit:    cfg.DailyGambleLimit,
		DailyLimitResetHour: cfg.DailyLimitResetHour,
	}
	discordBot, err := bot.New(botConfig, gamblingConfig, uowFactory, summonerClient, eventPublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Discord bot: %w", err)
	}
	log.Println("Discord bot initialized successfully")
	return discordBot, nil
}

// creates application-level handlers
func initializeApplicationHandlers(uowFactory application.UnitOfWorkFactory, discordBot *bot.Bot) *application.LoLHandlerImpl {
	log.Println("Initializing LoL handler...")
	lolHandler := application.NewLoLHandler(uowFactory, discordBot.GetDiscordPoster())
	log.Println("LoL handler initialized successfully")
	return lolHandler
}

// creates application-level workers
func initializeApplicationWorkers(uowFactory application.UnitOfWorkFactory, discordBot *bot.Bot) *application.DailyAwardsWorkerImpl {
	log.Println("Initializing daily awards worker...")
	guildDiscovery := bot.NewGuildDiscoveryService(discordBot.GetSession(), uowFactory)
	dailyAwardsWorker := application.NewDailyAwardsWorker(uowFactory, guildDiscovery, discordBot.GetDiscordPoster())
	log.Println("Daily awards worker initialized successfully")
	return dailyAwardsWorker
}

// registers all event subscriptions
func setupEventSubscriptions(natsClient *infrastructure.NATSClient, subjectMapper *infrastructure.EventSubjectMapper, uowFactory application.UnitOfWorkFactory, discordBot *bot.Bot, cfg *config.Config) error {
	log.Println("Initializing NATS event subscriber...")
	natsEventSubscriber := infrastructure.NewNATSEventSubscriber(natsClient, subjectMapper)

	// Create UserResolver
	userResolver := bot.NewUserResolver(discordBot.GetSession())

	// Register application-level subscriptions
	log.Println("Registering application event subscriptions...")
	if err := application.RegisterApplicationSubscriptions(
		natsEventSubscriber,
		uowFactory,
		discordBot.GetDiscordPoster(),
		userResolver,
	); err != nil {
		return fmt.Errorf("failed to register application subscriptions: %w", err)
	}

	// Register bot-level subscriptions
	log.Println("Registering bot event subscriptions...")
	if err := bot.RegisterBotSubscriptions(natsEventSubscriber, discordBot); err != nil {
		return fmt.Errorf("failed to register bot subscriptions: %w", err)
	}

	log.Println("All event subscriptions registered successfully")
	return nil
}

// starts all background services
func startBackgroundServices(ctx context.Context, cfg *config.Config, lolHandler *application.LoLHandlerImpl, dailyAwardsWorker *application.DailyAwardsWorkerImpl, discordBot *bot.Bot) (*infrastructure.MessageConsumer, []func()) {
	var cleanupFuncs []func()

	log.Printf("Initializing message consumer with NATS servers: %s...", cfg.NATSServers)
	messageConsumer := infrastructure.NewMessageConsumer(cfg.NATSServers, lolHandler)

	// Start message consumer in a goroutine
	go func() {
		if err := messageConsumer.Start(ctx); err != nil {
			log.Printf("Message consumer error: %v", err)
		}
	}()
	log.Println("Message consumer started successfully")

	// Start Discord bot workers
	log.Println("Starting Discord bot workers...")

	// Start group wager expiration worker
	groupWagerCleanup := discordBot.StartGroupWagerExpirationWorker(ctx)
	cleanupFuncs = append(cleanupFuncs, groupWagerCleanup)
	log.Println("Group wager expiration worker started")

	// Start daily awards worker
	dailyAwardsCleanup := dailyAwardsWorker.Start(ctx, cfg.DailyAwardsHour)
	cleanupFuncs = append(cleanupFuncs, dailyAwardsCleanup)
	discordBot.SetDailyAwardsWorkerCleanup(dailyAwardsCleanup)
	log.Printf("Daily awards worker started (notification at %02d:00 UTC)", cfg.DailyAwardsHour)

	return messageConsumer, cleanupFuncs
}

// handles graceful shutdown of all services
func performGracefulShutdown(
	messageConsumer *infrastructure.MessageConsumer,
	discordBot *bot.Bot,
	natsClient *infrastructure.NATSClient,
	summonerConn *grpc.ClientConn,
	db *database.DB,
	cleanupFuncs []func(),
) {
	log.Println("Shutting down services...")

	// Stop all workers
	log.Println("Stopping Discord bot workers...")
	for _, cleanup := range cleanupFuncs {
		cleanup()
	}

	// Stop message consumer
	log.Println("Stopping message consumer...")
	messageConsumer.Stop()

	// Close Discord bot connection
	if err := discordBot.Close(); err != nil {
		log.Printf("Error closing Discord bot: %v", err)
	}

	// Close NATS client
	if natsClient != nil {
		if err := natsClient.Close(); err != nil {
			log.Printf("Error closing NATS client: %v", err)
		}
	}

	// Close summoner tracking client connection
	if err := summonerConn.Close(); err != nil {
		log.Printf("Error closing summoner tracking client: %v", err)
	}

	// Give cleanup operations time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close database connection
	log.Println("Closing database connection...")
	db.Close()

	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded")
	case <-time.After(1 * time.Second):
		log.Println("Shutdown completed")
	}
}
