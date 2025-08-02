# Domain Layer - Clean Architecture Implementation

This package contains the pure domain layer following Clean Architecture principles.

## Structure

```
domain/
├── entities/       # Pure business entities with no external dependencies
├── events/         # Domain events for communicating state changes
├── interfaces/     # Repository and service contracts
├── services/       # Domain services for complex business logic (future)
└── interfaces.go   # Core domain interfaces
```

## Key Principles

### 1. Zero External Dependencies
The domain layer has NO imports outside of the Go standard library and itself. This ensures:
- Complete independence from infrastructure concerns
- Easy testing with no mocks needed for external dependencies
- Clear separation of business logic from technical implementation

### 2. Rich Domain Entities
All entities contain business logic and behavior, not just data:
- `User.CanAfford()` - Balance validation
- `Wager.IsActive()` - State validation
- `GroupWager.CanAcceptBets()` - Business rules
- `WordleScore.BasePoints()` - Scoring logic

### 3. Domain Events
Events represent important business occurrences:
- `BalanceChangeEvent` - When user balance changes
- `UserCreatedEvent` - When new user joins
- `WagerResolvedEvent` - When wager is settled
- `GroupWagerStateChangeEvent` - When group wager state transitions

### 4. Interface Segregation
Repository and service interfaces are focused and cohesive:
- `UserRepository` - User data access
- `WagerRepository` - Wager data access  
- `GroupWagerRepository` - Group wager operations
- `EventPublisher` - Event publishing contract

## Migration from Models

This domain layer extracts and purifies entities from the `models/` package:

### What Changed
- Removed all database tags (`db:"field_name"`)
- Added business methods and validation
- Extracted enums and value objects
- Created proper aggregate roots

### What Stayed
- All business logic preserved
- Field names and types maintained
- Existing behavior enhanced, not changed

## Next Steps (Future Phases)

1. **Phase 2**: Create application services that orchestrate domain operations
2. **Phase 3**: Update repository implementations to work with domain entities  
3. **Phase 4**: Migrate existing services to use domain layer
4. **Phase 5**: Remove old models package

## Usage Example

```go
// Creating a new user with business validation
user := &entities.User{
    DiscordID: 12345,
    Username: "player1", 
    Balance: 1000,
    AvailableBalance: 800,
}

// Business logic encapsulated in entity
if user.CanAfford(500) {
    // Place bet logic
}

// Domain events for side effects
event := events.BalanceChangeEvent{
    UserID: user.DiscordID,
    OldBalance: 1000,
    NewBalance: 500,
    ChangeAmount: -500,
    TransactionType: entities.TransactionTypeBetLoss,
}
```

This domain layer provides a solid foundation for clean, maintainable, and testable business logic.