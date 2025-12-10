# Gambler - Architecture Overview

A Discord bot gambling and economy ecosystem built as an event-driven microservices system.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              External Systems                                │
│   Discord API            NATS JetStream            PostgreSQL                │
└───────┬───────────────────────┬────────────────────────┬────────────────────┘
        │                       │                        │
┌───────▼───────────────────────▼────────────────────────▼────────────────────┐
│                          Discord Client (Go)                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                        Adapters (thin)                               │    │
│  │   Discord Bot (in/out)  │  NATS Pub/Sub  │  Repository Impls        │    │
│  └───────────────────────────────┬──────────────────────────────────────┘    │
│  ┌───────────────────────────────▼──────────────────────────────────────┐    │
│  │                     Application Layer                                 │    │
│  │   Command Handlers  │  Event Handlers  │  Unit of Work               │    │
│  └───────────────────────────────┬──────────────────────────────────────┘    │
│  ┌───────────────────────────────▼──────────────────────────────────────┐    │
│  │                       Domain Layer                                    │    │
│  │   Entities  │  Domain Services  │  Repository Interfaces             │    │
│  └──────────────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────────────┘
        │                       │
        │    Protocol Buffers   │
        │    ◄──────────────────┤
        │                       │
┌───────▼───────────────────────▼──────────────────────────────────────────────┐
│                          LoL Tracker (Python)                                │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                        Adapters                                       │   │
│  │   Riot API Client  │  NATS Publisher  │  SQLAlchemy Repos             │   │
│  └───────────────────────────────┬───────────────────────────────────────┘   │
│  ┌───────────────────────────────▼───────────────────────────────────────┐   │
│  │                     Application Layer                                  │   │
│  │                      Polling Service                                   │   │
│  └───────────────────────────────┬───────────────────────────────────────┘   │
│  ┌───────────────────────────────▼───────────────────────────────────────┐   │
│  │                       Core (Domain)                                    │   │
│  │   Entities  │  GameStateTransitionService                             │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Architecture Patterns

### Clean Architecture
Both services follow Clean Architecture with dependencies flowing inward toward the domain:

### Event-Driven Communication
Services communicate via NATS JetStream using protobuf:

### Unit of Work Pattern (Go service)
Ensures transactional consistency across database and events:
- Events buffered until transaction commits
- Rollbacks discard both database changes and pending events
- Guild-scoped instances for multi-tenant isolation

### Repository Pattern
- Domain defines interfaces
- Infrastructure provides implementations
- Enables testing with mocks without database dependencies

## Technology Stack

| Component | Discord Client | LoL Tracker |
|-----------|---------------|-------------|
| **Language** | Go 1.24 | Python 3.13 |
| **Database** | PostgreSQL (pgx) | PostgreSQL (SQLAlchemy) |
| **Messaging** | NATS JetStream | NATS JetStream |
| **External API** | Discord API (discordgo) | Riot Games API (httpx) |

## Data Flow Examples

### User Places a Bet
```
Discord Command → Command Handler → GamblingService → Repository
                                  ↓
                          Unit of Work commits
                                  ↓
                    Balance updated + Event published to NATS
```

### LoL Game State Change Detected
```
Riot API Poll → GameStateTransitionService → State change detected
                                           ↓
                              Event published to NATS
                                           ↓
              Discord Client receives event → Posts to channel
```

## Key Design Decisions

1. **Separate databases per service** - Each service owns its data
2. **Protocol Buffers for serialization** - Type-safe cross-language messaging
4. **Immutable balance history** - Full audit trail of all transactions
