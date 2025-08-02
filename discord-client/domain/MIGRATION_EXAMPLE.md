# Clean Architecture Migration Example

This document shows how to migrate existing code from the legacy service pattern to the new clean architecture.

## Example: Migrating a Handler

### Before (Legacy Pattern)

```go
// application/old_handler.go
func (h *OldHandler) HandleCommand(ctx context.Context, userID int64) error {
    // Create repositories using legacy pattern
    userRepo := repository.NewUserRepository(h.db)
    betRepo := repository.NewBetRepository(h.db)
    
    // Create service with repositories
    gamblingService := service.NewGamblingService(userRepo, betRepo, h.eventBus)
    
    // Use service directly
    result, err := gamblingService.PlaceBet(ctx, userID, 0.5, 1000)
    if err != nil {
        return err
    }
    
    // Work with models
    user := result.User // *models.User
    return h.sendResponse(user.Username, result.WonAmount)
}
```

### After (Clean Architecture)

```go
// application/new_handler.go
func (h *NewHandler) HandleCommand(ctx context.Context, userID int64) error {
    // Create Unit of Work
    uow, err := h.uowFactory.Create()
    if err != nil {
        return err
    }
    defer uow.Rollback()
    
    if err := uow.Begin(ctx); err != nil {
        return err
    }
    
    // Use domain interfaces through UoW
    user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
    if err != nil {
        return err
    }
    
    // Business logic using domain entities
    if user.Balance < 1000 {
        return errors.New("insufficient balance")
    }
    
    // Create domain entity
    bet := &entities.Bet{
        DiscordID:       userID,
        Amount:          1000,
        WinProbability:  0.5,
        CreatedAt:       time.Now(),
    }
    
    // Use domain service for business logic (optional)
    bettingService := services.NewBettingService(uow.UserRepository(), uow.BetRepository())
    result, err := bettingService.PlaceBet(ctx, bet)
    if err != nil {
        return err
    }
    
    // Commit transaction
    if err := uow.Commit(); err != nil {
        return err
    }
    
    // Work with entities
    return h.sendResponse(user.Username, result.WonAmount)
}
```

## Key Differences

### 1. Dependency Management

**Before:**
- Direct database dependencies
- Manual repository instantiation
- Services depend on concrete repositories

**After:**
- Unit of Work manages dependencies
- Repository interfaces from domain
- Transactional boundaries are clear

### 2. Data Types

**Before:**
```go
user := &models.User{...}  // Database models
```

**After:**
```go
user := &entities.User{...}  // Domain entities
```

### 3. Business Logic Location

**Before:**
- Business logic scattered across services
- Services tightly coupled to database models

**After:**
- Business logic in domain entities and services
- Clean separation between domain and infrastructure

## Migration Steps

### Step 1: Update Dependencies

**Before:**
```go
type Handler struct {
    db       *sql.DB
    eventbus EventBus
}
```

**After:**
```go
type Handler struct {
    uowFactory application.UnitOfWorkFactory
}
```

### Step 2: Replace Repository Calls

**Before:**
```go
userRepo := repository.NewUserRepository(h.db)
user, err := userRepo.GetByDiscordID(ctx, userID)
```

**After:**
```go
uow, err := h.uowFactory.Create()
// ... error handling and transaction setup
user, err := uow.UserRepository().GetByDiscordID(ctx, userID)
```

### Step 3: Convert Model Usage to Entity Usage

**Before:**
```go
bet := &models.Bet{
    DiscordID: userID,
    Amount:    amount,
    // ... other fields
}
err := betRepo.Create(ctx, bet)
```

**After:**
```go
bet := &entities.Bet{
    DiscordID: userID,
    Amount:    amount,
    // ... other fields
}
err := uow.BetRepository().Create(ctx, bet)
```

### Step 4: Handle Transactions

**Before:**
```go
// Manual transaction management
tx, err := db.Begin()
// ... operations
tx.Commit()
```

**After:**
```go
// Unit of Work handles transactions
uow, err := h.uowFactory.Create()
defer uow.Rollback()

err = uow.Begin(ctx)
// ... operations
err = uow.Commit()
```

## Domain Service Example

For complex business logic, create domain services:

```go
// domain/services/betting_service.go
package services

type BettingService struct {
    userRepo interfaces.UserRepository
    betRepo  interfaces.BetRepository
}

func NewBettingService(userRepo interfaces.UserRepository, betRepo interfaces.BetRepository) *BettingService {
    return &BettingService{
        userRepo: userRepo,
        betRepo:  betRepo,
    }
}

func (s *BettingService) PlaceBet(ctx context.Context, bet *entities.Bet) (*entities.BetResult, error) {
    // Domain business logic here
    user, err := s.userRepo.GetByDiscordID(ctx, bet.DiscordID) 
    if err != nil {
        return nil, err
    }
    
    // Validate business rules
    if err := bet.Validate(); err != nil {
        return nil, err
    }
    
    if user.Balance < bet.Amount {
        return nil, entities.ErrInsufficientBalance
    }
    
    // Execute bet logic
    result := bet.Execute()
    
    // Update user balance
    user.UpdateBalance(result.NewBalance)
    if err := s.userRepo.UpdateBalance(ctx, user.DiscordID, user.Balance); err != nil {
        return nil, err
    }
    
    // Save bet record
    if err := s.betRepo.Create(ctx, bet); err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## Testing Benefits

### Before
```go
func TestOldHandler(t *testing.T) {
    // Need real database or complex mocking
    db := setupTestDB()
    handler := &OldHandler{db: db}
    
    // Test involves database operations
    err := handler.HandleCommand(ctx, 123)
    assert.NoError(t, err)
}
```

### After
```go
func TestNewHandler(t *testing.T) {
    // Mock the UoW factory
    mockUoW := &mocks.MockUnitOfWork{}
    mockUserRepo := &mocks.MockUserRepository{}
    
    mockUoW.On("UserRepository").Return(mockUserRepo)
    mockUserRepo.On("GetByDiscordID", mock.Anything, int64(123)).Return(&entities.User{Balance: 5000}, nil)
    
    factory := &mocks.MockUoWFactory{}
    factory.On("Create").Return(mockUoW, nil)
    
    handler := &NewHandler{uowFactory: factory}
    
    // Pure unit test - no database needed
    err := handler.HandleCommand(ctx, 123)
    assert.NoError(t, err)
}
```

## Common Pitfalls

### 1. Forgetting Transaction Management
Always use the Unit of Work pattern for transactional operations:

```go
// Wrong - no transaction
user, _ := uow.UserRepository().GetByDiscordID(ctx, userID)
user.Balance += 1000
uow.UserRepository().UpdateBalance(ctx, userID, user.Balance)

// Right - proper transaction
uow, _ := h.uowFactory.Create()
defer uow.Rollback()
uow.Begin(ctx)

user, _ := uow.UserRepository().GetByDiscordID(ctx, userID)
user.Balance += 1000
uow.UserRepository().UpdateBalance(ctx, userID, user.Balance)

uow.Commit()
```

### 2. Mixing Models and Entities
Don't mix the old models with new entities:

```go
// Wrong - mixing types
user := uow.UserRepository().GetByDiscordID(ctx, userID) // returns *entities.User
oldService.DoSomething(user) // expects *models.User

// Right - consistent entity usage
user := uow.UserRepository().GetByDiscordID(ctx, userID) // returns *entities.User
domainService.DoSomething(user) // expects *entities.User
```

### 3. Business Logic in Handlers
Keep handlers thin, move business logic to domain services:

```go
// Wrong - business logic in handler
func (h *Handler) HandleCommand(ctx context.Context, userID int64, amount int64) error {
    user, _ := uow.UserRepository().GetByDiscordID(ctx, userID)
    if user.Balance < amount {
        return errors.New("insufficient balance")
    }
    if amount > 10000 {
        return errors.New("amount too large")
    }
    // ... more business logic
}

// Right - business logic in domain service
func (h *Handler) HandleCommand(ctx context.Context, userID int64, amount int64) error {
    bettingService := services.NewBettingService(uow.UserRepository(), uow.BetRepository())
    return bettingService.PlaceBet(ctx, userID, amount)
}
```

This migration pattern ensures better testability, maintainability, and follows clean architecture principles.