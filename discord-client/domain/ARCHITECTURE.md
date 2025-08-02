# Clean Architecture Implementation

This document describes the clean architecture implementation in the gambler discord-client codebase.

## Overview

The codebase has been restructured to follow Clean Architecture principles, separating concerns into distinct layers with clear dependencies flowing inward toward the domain.

## Architecture Layers

### 1. Domain Layer (`domain/`)

The innermost layer containing business logic and core entities.

**Key Components:**
- `entities/` - Core business entities (User, Wager, Bet, etc.)
- `interfaces/` - Abstract interfaces for repositories and services
- `services/` - Domain business logic services
- `events/` - Domain events

**Responsibilities:**
- Define core business entities
- Contain business rules and invariants
- Define abstract interfaces for external dependencies
- Implement domain services that contain business logic

**Dependencies:** None (depends only on Go standard library)

### 2. Application Layer (`application/`)

Orchestrates use cases and coordinates between domain and infrastructure.

**Key Components:**
- `unit_of_work.go` - Transactional coordination interface
- `*_handler.go` - Discord command handlers
- `*_worker.go` - Background job handlers

**Responsibilities:**
- Coordinate use cases
- Handle Discord interactions
- Manage transactions via Unit of Work pattern
- Convert between domain entities and external formats

**Dependencies:** Domain layer only

### 3. Infrastructure Layer (`infrastructure/`)

Provides implementations for external concerns.

**Key Components:**
- `adapters/` - Implementation adapters for domain interfaces
  - `repositories/` - Database repository implementations
  - `*_adapter.go` - Entity/model conversion adapters
- `unit_of_work.go` - Concrete Unit of Work implementation

**Responsibilities:**
- Implement domain interfaces
- Handle database operations
- Manage external service integrations
- Convert between entities and database models

**Dependencies:** Application and Domain layers

### 4. Legacy Service Layer (`service/`)

**Status:** DEPRECATED - Being phased out

Contains legacy services that work with models instead of entities. These are being gradually migrated to use the clean architecture.

## Key Patterns

### Repository Pattern

Domain interfaces define repository contracts:

```go
// domain/interfaces/repositories.go
type UserRepository interface {
    GetByDiscordID(ctx context.Context, discordID int64) (*entities.User, error)
    Create(ctx context.Context, discordID int64, username string, initialBalance int64) (*entities.User, error)
    // ...
}
```

Infrastructure provides implementations:

```go
// infrastructure/adapters/repositories/user_repository_adapter.go
type userRepositoryAdapter struct {
    modelRepo repository.UserRepository // Legacy repository
    adapter   *UserAdapter              // Entity/model converter
}
```

### Adapter Pattern

Adapters convert between domain entities and database models:

```go
// infrastructure/adapters/user_adapter.go
func (a *UserAdapter) ToEntity(model *models.User) *entities.User {
    return &entities.User{
        DiscordID: model.DiscordID,
        Username:  model.Username,
        Balance:   model.Balance,
        CreatedAt: model.CreatedAt,
        UpdatedAt: model.UpdatedAt,
    }
}
```

### Unit of Work Pattern

Coordinates transactions across multiple repositories:

```go
// application/unit_of_work.go
type UnitOfWork interface {
    Begin(ctx context.Context) error
    Commit() error
    Rollback() error
    
    UserRepository() interfaces.UserRepository
    WagerRepository() interfaces.WagerRepository
    // ... other repositories
}
```

## Data Flow

1. **Inbound:** Discord API → Application Handlers → Domain Services → Repository Interfaces
2. **Outbound:** Repository Implementations → Database Models → Entity Adapters → Domain Entities

## Migration Guide

### For New Features

1. Define entities in `domain/entities/`
2. Define repository interfaces in `domain/interfaces/`
3. Implement business logic in `domain/services/` (optional)
4. Create application handlers in `application/`
5. Implement repository adapters in `infrastructure/adapters/repositories/`

### For Existing Code

1. **Phase 1:** Extract entities from models
2. **Phase 2:** Define domain interfaces
3. **Phase 3:** Create adapter implementations
4. **Phase 4:** Update application layer to use domain interfaces
5. **Phase 5:** Deprecate legacy service layer

## Benefits

### Testability
- Domain logic is isolated and easily testable
- Repository interfaces can be mocked
- No database dependencies in business logic tests

### Maintainability
- Clear separation of concerns
- Dependencies flow inward (Dependency Inversion Principle)
- Business logic is independent of external frameworks

### Flexibility
- Easy to swap database implementations
- Business logic can be reused across different interfaces
- External services can be easily mocked or replaced

## Current Status

- ✅ Domain entities defined
- ✅ Domain interfaces established
- ✅ Repository adapters implemented
- ✅ Unit of Work pattern in place
- 🔄 Application layer partially migrated
- 🔄 Legacy service layer being deprecated

## Usage Examples

### Creating a New Repository

1. Define the interface in `domain/interfaces/repositories.go`:
```go
type MyEntityRepository interface {
    Create(ctx context.Context, entity *entities.MyEntity) error
    GetByID(ctx context.Context, id int64) (*entities.MyEntity, error)
}
```

2. Create the adapter in `infrastructure/adapters/repositories/`:
```go
type myEntityRepositoryAdapter struct {
    modelRepo repository.MyEntityRepository
    adapter   *MyEntityAdapter
}

func (r *myEntityRepositoryAdapter) Create(ctx context.Context, entity *entities.MyEntity) error {
    model := r.adapter.ToModel(entity)
    return r.modelRepo.Create(ctx, model)
}
```

3. Add to Unit of Work:
```go
// In UnitOfWork interface
MyEntityRepository() interfaces.MyEntityRepository

// In implementation
func (uow *unitOfWork) MyEntityRepository() interfaces.MyEntityRepository {
    return repositories.NewMyEntityRepositoryAdapter(
        repository.NewMyEntityRepository(uow.db),
        adapters.NewMyEntityAdapter(),
    )
}
```

### Using in Application Layer

```go
func (h *MyHandler) HandleCommand(ctx context.Context) error {
    uow, err := h.uowFactory.Create()
    if err != nil {
        return err
    }
    defer uow.Rollback()

    if err := uow.Begin(ctx); err != nil {
        return err
    }

    // Use domain repositories
    entity, err := uow.MyEntityRepository().GetByID(ctx, 123)
    if err != nil {
        return err
    }

    // Modify entity using domain logic
    entity.DoSomething()

    // Save changes
    if err := uow.MyEntityRepository().Update(ctx, entity); err != nil {
        return err
    }

    return uow.Commit()
}
```

## Future Improvements

1. Complete migration of application layer
2. Remove legacy service layer
3. Add domain services for complex business logic
4. Implement domain events pattern
5. Add integration tests for adapters