package services

import (
	"gambler/discord-client/domain/interfaces"
	"gambler/discord-client/domain/testhelpers"
	"testing"

	"gambler/discord-client/config"

	"github.com/stretchr/testify/assert"
)

// Test utilities

func createTestGroupWagerService() (interfaces.GroupWagerService, *testhelpers.MockUserRepository, *testhelpers.MockGroupWagerRepository, *testhelpers.MockBalanceHistoryRepository, *testhelpers.MockEventPublisher) {
	config.SetTestConfig(config.NewTestConfig())

	mockUserRepo := new(testhelpers.MockUserRepository)
	mockGroupWagerRepo := new(testhelpers.MockGroupWagerRepository)
	mockBalanceHistoryRepo := new(testhelpers.MockBalanceHistoryRepository)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	service := NewGroupWagerService(mockGroupWagerRepo, mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)
	return service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher
}

// Tests

func TestGroupWagerService_IsResolver(t *testing.T) {

	t.Run("user is resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Test config sets 999999 as the default resolver ID
		assert.True(t, service.IsResolver(999999))
		// Non-resolvers
		assert.False(t, service.IsResolver(111111))
		assert.False(t, service.IsResolver(222222))
	})

	t.Run("user is not resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Test config sets 999999 as the default resolver ID
		// These should not be resolvers
		assert.False(t, service.IsResolver(444444))
		assert.False(t, service.IsResolver(555555))
		assert.False(t, service.IsResolver(0))
	})

	t.Run("test resolver constant", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Test config sets 999999 as the default resolver ID
		// TestResolverID constant should be a resolver
		assert.True(t, service.IsResolver(TestResolverID))
	})
}
