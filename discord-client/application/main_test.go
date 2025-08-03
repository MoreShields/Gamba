package application_test

import (
	"os"
	"testing"

	"gambler/discord-client/config"
)

func TestMain(m *testing.M) {
	// Set up test config once for all tests
	testConfig := config.NewTestConfig()
	testConfig.DiscordToken = "test-token" // Add a test token to prevent validation errors
	config.SetTestConfig(testConfig)

	// Ensure config is loaded before running tests
	_ = config.Get()

	// Run tests
	code := m.Run()

	// Exit with test result code
	os.Exit(code)
}
