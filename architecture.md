@startuml Gambler Discord Bot Architecture

!theme plain

' Application Layer
package "Application Layer" {
  [main.go] as Main
  
  package "cmd" {
    [run.go] as Run
  }
  
  package "config" {
    [config.go] as Config
  }
}

' Bot Layer
package "Bot Layer" {
  [bot.go] as Bot
  [bet_handler.go] as BetHandler
  [wager_handler.go] as WagerHandler
  [group_wager_handler.go] as GroupWagerHandler
  [stats_handler.go] as StatsHandler
  [user_helpers.go] as UserHelpers
  [*_embeds.go] as EmbedFormatters
  [*_components.go] as ComponentBuilders
}

' Service Layer
package "Service Layer" {
  interface "UserService" as IUserService
  interface "GamblingService" as IGamblingService
  interface "TransferService" as ITransferService
  interface "WagerService" as IWagerService
  interface "GroupWagerService" as IGroupWagerService
  interface "StatsService" as IStatsService
  
  [UserService] as UserServiceImpl
  [GamblingService] as GamblingServiceImpl
  [TransferService] as TransferServiceImpl
  [WagerService] as WagerServiceImpl 
  [GroupWagerService] as GroupWagerServiceImpl
  [StatsService] as StatsServiceImpl
  
  interface "UnitOfWorkFactory" as IUnitOfWorkFactory
  interface "UnitOfWork" as IUnitOfWork
}

' Repository Layer
package "Repository Layer" {
  interface "UserRepository" as IUserRepository
  interface "BalanceHistoryRepository" as IBalanceHistoryRepository
  interface "BetRepository" as IBetRepository
  interface "WagerRepository" as IWagerRepository
  interface "WagerVoteRepository" as IWagerVoteRepository
  interface "GroupWagerRepository" as IGroupWagerRepository
  
  [UserRepository] as UserRepositoryImpl
  [BalanceHistoryRepository] as BalanceHistoryRepositoryImpl
  [BetRepository] as BetRepositoryImpl
  [WagerRepository] as WagerRepositoryImpl
  [WagerVoteRepository] as WagerVoteRepositoryImpl
  [GroupWagerRepository] as GroupWagerRepositoryImpl
  
  [UnitOfWorkFactory] as UnitOfWorkFactoryImpl
  [UnitOfWork] as UnitOfWorkImpl
}

' Data Layer
package "Data Layer" {
  [database/connection.go] as DBConnection
  [database/migrate.go] as DBMigrate
  [database/transaction.go] as DBTransaction
  [(PostgreSQL)] as Database
}

' Models
package "Models" {
  [User] as UserModel
  [BalanceHistory] as BalanceHistoryModel
  [Bet] as BetModel
  [Wager] as WagerModel
  [WagerVote] as WagerVoteModel
  [GroupWager] as GroupWagerModel
  [*Stats] as StatsModels
}

' Events
package "Events" {
  [events.go] as EventsHandler
  [TransactionalBus] as TransactionalBus
}

' External Dependencies
package "External" {
  [Discord API] as DiscordAPI
  [Cron Scheduler] as CronScheduler
}

' Relationships - Application Layer
Main --> Run
Run --> Config
Run --> Bot
Run --> CronScheduler

' Relationships - Bot Layer
Bot --> IUserService
Bot --> IGamblingService
Bot --> ITransferService
Bot --> IWagerService
Bot --> IGroupWagerService
Bot --> IStatsService

BetHandler --> IGamblingService
WagerHandler --> IWagerService
GroupWagerHandler --> IGroupWagerService
StatsHandler --> IStatsService
UserHelpers --> IUserService

Bot --> DiscordAPI

' Relationships - Service Layer
IUserService <|-- UserServiceImpl
IGamblingService <|-- GamblingServiceImpl
ITransferService <|-- TransferServiceImpl
IWagerService <|-- WagerServiceImpl
IGroupWagerService <|-- GroupWagerServiceImpl
IStatsService <|-- StatsServiceImpl

UserServiceImpl --> IUnitOfWorkFactory
GamblingServiceImpl --> IUnitOfWorkFactory
TransferServiceImpl --> IUnitOfWorkFactory
WagerServiceImpl --> IUnitOfWorkFactory
GroupWagerServiceImpl --> IUnitOfWorkFactory
StatsServiceImpl --> IUnitOfWorkFactory

IUnitOfWorkFactory <|-- UnitOfWorkFactoryImpl
IUnitOfWork <|-- UnitOfWorkImpl

' Relationships - Repository Layer
IUserRepository <|-- UserRepositoryImpl
IBalanceHistoryRepository <|-- BalanceHistoryRepositoryImpl
IBetRepository <|-- BetRepositoryImpl
IWagerRepository <|-- WagerRepositoryImpl
IWagerVoteRepository <|-- WagerVoteRepositoryImpl
IGroupWagerRepository <|-- GroupWagerRepositoryImpl

UnitOfWorkImpl --> IUserRepository
UnitOfWorkImpl --> IBalanceHistoryRepository
UnitOfWorkImpl --> IBetRepository
UnitOfWorkImpl --> IWagerRepository
UnitOfWorkImpl --> IWagerVoteRepository
UnitOfWorkImpl --> IGroupWagerRepository

' Relationships - Data Layer
UserRepositoryImpl --> DBConnection
BalanceHistoryRepositoryImpl --> DBConnection
BetRepositoryImpl --> DBConnection
WagerRepositoryImpl --> DBConnection
WagerVoteRepositoryImpl --> DBConnection
GroupWagerRepositoryImpl --> DBConnection

UnitOfWorkImpl --> DBConnection
UnitOfWorkImpl --> DBTransaction
DBConnection --> Database
DBMigrate --> Database

' Relationships - Models
UserRepositoryImpl --> UserModel
BalanceHistoryRepositoryImpl --> BalanceHistoryModel
BetRepositoryImpl --> BetModel
WagerRepositoryImpl --> WagerModel
WagerVoteRepositoryImpl --> WagerVoteModel
GroupWagerRepositoryImpl --> GroupWagerModel
StatsServiceImpl --> StatsModels

' Relationships - Events
UnitOfWorkImpl --> TransactionalBus
TransactionalBus --> EventsHandler

' Transaction Flow
note right of IUnitOfWork
  Unit of Work Pattern:
  1. Service creates UnitOfWork
  2. Begin transaction
  3. Get repositories
  4. Perform operations
  5. Commit/Rollback
  6. Publish events
end note

' Dependency Injection
note right of Run
  Dependency Injection:
  1. Load configuration
  2. Create DB connection
  3. Create repository implementations
  4. Create service implementations
  5. Wire bot with services
  6. Start application
end note

' Architecture Layers
note top of "Application Layer"
  Entry point and configuration
end note

note top of "Bot Layer"
  Discord interaction handlers
end note

note top of "Service Layer"
  Business logic and transactions
end note

note top of "Repository Layer"
  Data access and persistence
end note

note top of "Data Layer"
  Database connection and management
end note

@enduml
