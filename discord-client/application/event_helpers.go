package application

import (
	"fmt"
	"reflect"

	"gambler/discord-client/domain/events"
)

// AssertEventType safely asserts an event to a specific type with detailed error messages
func AssertEventType[T events.Event](event interface{}, expectedTypeName string) (T, error) {
	var zero T

	// Try direct type assertion first
	if e, ok := event.(T); ok {
		return e, nil
	}

	// Get actual type information for error message
	actualType := fmt.Sprintf("%T", event)

	// If event implements Event interface, get its Type() method result
	var eventTypeStr string
	if e, ok := event.(events.Event); ok {
		eventTypeStr = string(e.Type())
	}

	// Build detailed error message
	errMsg := fmt.Sprintf("event type assertion failed: expected %s, got %s", expectedTypeName, actualType)
	if eventTypeStr != "" {
		errMsg += fmt.Sprintf(" (event.Type()=%s)", eventTypeStr)
	}

	// Add value information if it's a pointer to help debug nil issues
	if reflect.ValueOf(event).Kind() == reflect.Ptr {
		if reflect.ValueOf(event).IsNil() {
			errMsg += " (event is nil)"
		} else {
			// Try to get the underlying type
			elemType := reflect.TypeOf(event).Elem()
			errMsg += fmt.Sprintf(" (pointer to %s)", elemType.String())
		}
	}

	return zero, fmt.Errorf("%s", errMsg)
}
