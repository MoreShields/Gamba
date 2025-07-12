package betting

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gambler/service"

	"github.com/bwmarrin/discordgo"
)

// GamblingConfig holds gambling-specific configuration
type GamblingConfig struct {
	DailyGambleLimit    int64
	DailyLimitResetHour int
}

// Feature represents the betting feature
type Feature struct {
	config     *GamblingConfig
	uowFactory service.UnitOfWorkFactory
}

// New creates a new betting feature instance
func New(config *GamblingConfig, uowFactory service.UnitOfWorkFactory) *Feature {
	f := &Feature{
		config:     config,
		uowFactory: uowFactory,
	}

	// Start session cleanup
	go f.startSessionCleanup()

	return f
}

// HandleCommand handles the /gamble command
func (f *Feature) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f.handleGamble(s, i)
}

// HandleInteraction handles betting-related component interactions
func (f *Feature) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		f.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		f.handleModalSubmit(s, i)
	}
}

// startSessionCleanup runs periodic cleanup of old bet sessions
func (f *Feature) startSessionCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanupSessions()
	}
}

// createUnitOfWork creates and begins a guild-scoped unit of work from a Discord interaction
func (f *Feature) createUnitOfWork(ctx context.Context, i *discordgo.InteractionCreate) (service.UnitOfWork, error) {
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing guild ID %s: %w", i.GuildID, err)
	}

	uow := f.uowFactory.CreateForGuild(guildID)
	if err := uow.Begin(ctx); err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}

	return uow, nil
}
