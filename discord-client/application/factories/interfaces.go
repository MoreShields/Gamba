package factories

import (
	"context"
	"gambler/discord-client/application"
	"gambler/discord-client/application/dto"
	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/services"
)

// HandlerFactory creates application handlers with their dependencies
type HandlerFactory interface {
	// CreateLoLHandler creates a new LoL event handler
	CreateLoLHandler() application.LoLEventHandler
	
	// CreateWagerStateEventHandler creates a new wager state event handler
	CreateWagerStateEventHandler() application.WagerStateEventHandler
	
	// CreateWordleHandler creates a new Wordle handler
	CreateWordleHandler() interface{} // TODO: Define proper interface when needed
}

// DTOFactory creates DTOs from domain entities
type DTOFactory interface {
	// CreateHouseWagerPostDTO creates a house wager post DTO from domain entities
	CreateHouseWagerPostDTO(ctx context.Context, wager *entities.GroupWager, options []*entities.GroupWagerOption, participants []*entities.GroupWagerParticipant) dto.HouseWagerPostDTO
	
	// CreateDailyAwardsPostDTO creates a daily awards post DTO from domain entities
	CreateDailyAwardsPostDTO(ctx context.Context, awards []services.DailyAward, guildInfo dto.GuildChannelInfo) dto.DailyAwardsPostDTO
	
	// CreateGameStartedDTO creates a game started DTO from raw event data
	CreateGameStartedDTO(gameData map[string]interface{}) dto.GameStartedDTO
	
	// CreateGameEndedDTO creates a game ended DTO from raw event data
	CreateGameEndedDTO(gameData map[string]interface{}) dto.GameEndedDTO
}

// ServiceFactory creates application services with their dependencies
type ServiceFactory interface {
	// CreateDailyAwardsWorker creates a new daily awards worker service
	CreateDailyAwardsWorker() interface{} // TODO: Define proper WorkerService interface when needed
	
	// CreateGuildDiscoveryService creates a new guild discovery service
	CreateGuildDiscoveryService() application.GuildDiscoveryService
}