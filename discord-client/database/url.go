package database

import (
	"fmt"
	"strings"
)

// ConstructDatabaseURL constructs a complete database URL from base URL and database name
// This function:
// - Combines base URL with database name
// - Automatically adds sslmode=disable if not present
// - Handles existing query parameters correctly
func ConstructDatabaseURL(baseURL, databaseName string) string {
	// If DATABASE_NAME is not set, return the base URL as-is
	if databaseName == "" {
		return baseURL
	}
	
	// Remove trailing slash from base URL
	baseURL = strings.TrimRight(baseURL, "/")
	var databaseURL string
	
	// Check if there are existing query parameters
	if strings.Contains(baseURL, "?") {
		// Insert database name before the query parameters
		parts := strings.SplitN(baseURL, "?", 2)
		databaseURL = fmt.Sprintf("%s/%s?%s", parts[0], databaseName, parts[1])
	} else {
		// No query parameters - simply append database name
		databaseURL = fmt.Sprintf("%s/%s", baseURL, databaseName)
	}
	
	// Add sslmode=disable if not already present
	if !strings.Contains(databaseURL, "sslmode=") {
		separator := "&"
		if !strings.Contains(databaseURL, "?") {
			separator = "?"
		}
		databaseURL = fmt.Sprintf("%s%ssslmode=disable", databaseURL, separator)
	}
	
	return databaseURL
}