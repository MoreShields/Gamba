package services

import (
	"context"
	"testing"

	"gambler/discord-client/domain/entities"
	"gambler/discord-client/domain/interfaces"
)

// GroupWagerTestFixture provides a complete test environment for group wager service tests
type GroupWagerTestFixture struct {
	T          *testing.T
	Ctx        context.Context
	Service    interfaces.GroupWagerService
	Mocks      *TestMocks
	Helper     *MockHelper
	Assertions *AssertionHelper
}

// NewGroupWagerTestFixture creates a new test fixture with all dependencies configured
func NewGroupWagerTestFixture(t *testing.T) *GroupWagerTestFixture {
	// Setup test config
	SetupTestConfig(t)

	// Create mocks
	mocks := NewTestMocks()

	// Create service
	service := NewGroupWagerService(
		mocks.GroupWagerRepo,
		mocks.UserRepo,
		mocks.BalanceHistoryRepo,
		mocks.EventPublisher,
	)

	return &GroupWagerTestFixture{
		T:          t,
		Ctx:        context.Background(),
		Service:    service,
		Mocks:      mocks,
		Helper:     NewMockHelper(mocks),
		Assertions: NewAssertionHelper(t),
	}
}

// SetResolvers configures the resolver IDs for the test
func (f *GroupWagerTestFixture) SetResolvers(resolverIDs ...int64) {
	// Note: Can't access internal config anymore - this would need to be done differently
}

// GetServiceConfig returns the service config for advanced test scenarios
// Note: This is no longer available since the service implementation is now private
// func (f *GroupWagerTestFixture) GetServiceConfig() *config.Config {
//	return f.Service.(*groupWagerService).config
// }

// AssertAllMocks verifies all mock expectations were met
func (f *GroupWagerTestFixture) AssertAllMocks() {
	f.Mocks.AssertAllExpectations(f.T)
}

// Reset clears all mock expectations for reuse in sub-tests
func (f *GroupWagerTestFixture) Reset() {
	// Create fresh mocks
	f.Mocks = NewTestMocks()
	f.Helper = NewMockHelper(f.Mocks)

	// Recreate service with new mocks
	f.Service = NewGroupWagerService(
		f.Mocks.GroupWagerRepo,
		f.Mocks.UserRepo,
		f.Mocks.BalanceHistoryRepo,
		f.Mocks.EventPublisher,
	)
}

// WithScenario sets up mocks for a complete test scenario
func (f *GroupWagerTestFixture) WithScenario(scenario *GroupWagerScenario) *GroupWagerTestFixture {
	// Setup user mocks
	for _, user := range scenario.Users {
		f.Helper.ExpectUserLookup(user.DiscordID, user)
	}

	// Setup wager mocks
	if scenario.Wager != nil {
		f.Helper.ExpectWagerLookup(scenario.Wager.ID, scenario.Wager)
		f.Helper.ExpectWagerDetailLookup(scenario.Wager.ID, &entities.GroupWagerDetail{
			Wager:        scenario.Wager,
			Options:      scenario.Options,
			Participants: scenario.Participants,
		})
	}

	return f
}

// ExpectSuccess is a helper for tests that should succeed
func (f *GroupWagerTestFixture) ExpectSuccess(fn func() error) {
	err := fn()
	f.Assertions.AssertNoError(err)
	f.AssertAllMocks()
}

// ExpectError is a helper for tests that should fail with specific error
func (f *GroupWagerTestFixture) ExpectError(fn func() error, expectedError string) {
	err := fn()
	f.Assertions.AssertValidationError(err, expectedError)
	f.AssertAllMocks()
}

// Nil asserts that a value is nil
func (f *GroupWagerTestFixture) Nil(value interface{}) {
	// Use reflection to check for typed nil
	if value == nil {
		return
	}

	// Check for typed nil (e.g., (*Type)(nil))
	switch v := value.(type) {
	case *entities.GroupWagerDetail:
		if v != nil {
			f.T.Errorf("expected nil, got %v", value)
		}
	default:
		f.T.Errorf("expected nil, got %v", value)
	}
}

// NotNil asserts that a value is not nil
func (f *GroupWagerTestFixture) NotNil(value interface{}) {
	if value == nil {
		f.T.Errorf("expected non-nil value")
	}
}

// Equal asserts that two values are equal
func (f *GroupWagerTestFixture) Equal(expected, actual interface{}) {
	if expected != actual {
		f.T.Errorf("expected %v, got %v", expected, actual)
	}
}

// Len asserts that a slice has the expected length
func (f *GroupWagerTestFixture) Len(slice interface{}, expectedLen int) {
	switch v := slice.(type) {
	case []string:
		if len(v) != expectedLen {
			f.T.Errorf("expected length %d, got %d", expectedLen, len(v))
		}
	case []*entities.GroupWagerOption:
		if len(v) != expectedLen {
			f.T.Errorf("expected length %d, got %d", expectedLen, len(v))
		}
	default:
		f.T.Errorf("Len not implemented for type %T", slice)
	}
}
