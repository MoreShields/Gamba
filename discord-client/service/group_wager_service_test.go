package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test utilities

func createTestGroupWagerService() (GroupWagerService, *MockUserRepository, *MockGroupWagerRepository, *MockBalanceHistoryRepository, *MockEventPublisher) {
	mockUserRepo := new(MockUserRepository)
	mockGroupWagerRepo := new(MockGroupWagerRepository)
	mockBalanceHistoryRepo := new(MockBalanceHistoryRepository)
	mockEventPublisher := new(MockEventPublisher)

	service := NewGroupWagerService(mockGroupWagerRepo, mockUserRepo, mockBalanceHistoryRepo, mockEventPublisher)
	return service, mockUserRepo, mockGroupWagerRepo, mockBalanceHistoryRepo, mockEventPublisher
}

// Tests

func TestGroupWagerService_IsResolver(t *testing.T) {
	t.Run("user is resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{111111, 222222, 333333}

		// Test
		assert.True(t, service.IsResolver(111111))
		assert.True(t, service.IsResolver(222222))
		assert.True(t, service.IsResolver(333333))
	})

	t.Run("user is not resolver", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Set resolver IDs
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{111111, 222222, 333333}

		// Test
		assert.False(t, service.IsResolver(444444))
		assert.False(t, service.IsResolver(555555))
		assert.False(t, service.IsResolver(0))
	})

	t.Run("empty resolver list", func(t *testing.T) {
		// Setup
		service, _, _, _, _ := createTestGroupWagerService()

		// Set empty resolver list
		service.(*groupWagerService).config.ResolverDiscordIDs = []int64{}

		// Test
		assert.False(t, service.IsResolver(111111))
		assert.False(t, service.IsResolver(222222))
	})
}
