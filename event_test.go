package goat

import (
	"reflect"
	"testing"
)

type testEventWithPointer struct {
	Event
	ptr *testStruct
}

type testStruct struct {
	value int
}

func TestCloneEvent(t *testing.T) {
	tests := []struct {
		name     string
		original AbstractEvent
	}{
		{
			name:     "testEvent",
			original: &testEvent{Value: 42},
		},
		{
			name:     "EntryEvent",
			original: &EntryEvent{},
		},
		{
			name:     "ExitEvent",
			original: &ExitEvent{},
		},
		{
			name:     "HaltEvent",
			original: &HaltEvent{},
		},
		{
			name:     "TransitionEvent",
			original: &TransitionEvent{To: &testState{Name: "target"}},
		},
		{
			name:     "testEventWithPointer",
			original: &testEventWithPointer{ptr: &testStruct{value: 100}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := cloneEvent(tt.original)

			// Verify pointer addresses are different
			if reflect.ValueOf(tt.original).Pointer() == reflect.ValueOf(cloned).Pointer() {
				t.Errorf("Expected different pointer addresses, but got the same: %p", tt.original)
			}

			// Verify types are the same
			if reflect.TypeOf(tt.original) != reflect.TypeOf(cloned) {
				t.Errorf("Expected same type, but got different: %T vs %T", tt.original, cloned)
			}

			if !reflect.DeepEqual(tt.original, cloned) {
				t.Errorf("Expected original and cloned events to be equal, but they are not: %v vs %v", tt.original, cloned)
			}
		})
	}
}

func TestSameEvent(t *testing.T) {
	tests := []struct {
		name     string
		event1   AbstractEvent
		event2   AbstractEvent
		expected bool
	}{
		{
			name:     "same testEvent types",
			event1:   &testEvent{Value: 1},
			event2:   &testEvent{Value: 2},
			expected: true,
		},
		{
			name:     "same EntryEvent types",
			event1:   &EntryEvent{},
			event2:   &EntryEvent{},
			expected: true,
		},
		{
			name:     "different types",
			event1:   &testEvent{},
			event2:   &EntryEvent{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sameEvent(tt.event1, tt.event2)
			if result != tt.expected {
				t.Errorf("sameEvent(%T, %T) = %v, expected %v", tt.event1, tt.event2, result, tt.expected)
			}
		})
	}
}

func TestGetEventName(t *testing.T) {
	tests := []struct {
		name     string
		event    AbstractEvent
		expected string
	}{
		{
			name:     "testEvent",
			event:    &testEvent{},
			expected: "testEvent",
		},
		{
			name:     "EntryEvent",
			event:    &EntryEvent{},
			expected: "EntryEvent",
		},
		{
			name:     "ExitEvent",
			event:    &ExitEvent{},
			expected: "ExitEvent",
		},
		{
			name:     "TransitionEvent",
			event:    &TransitionEvent{},
			expected: "TransitionEvent",
		},
		{
			name:     "HaltEvent",
			event:    &HaltEvent{},
			expected: "HaltEvent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEventName(tt.event)
			if result != tt.expected {
				t.Errorf("getEventName(%T) = %s, expected %s", tt.event, result, tt.expected)
			}
		})
	}
}

func TestGetEventDetails(t *testing.T) {
	tests := []struct {
		name     string
		event    AbstractEvent
		expected string
	}{
		{
			name:     "testEvent with value",
			event:    &testEvent{Value: 42},
			expected: "{Name:Value,Type:int,Value:42}",
		},
		{
			name:     "EntryEvent no fields",
			event:    &EntryEvent{},
			expected: noFieldsMessage,
		},
		{
			name:     "TransitionEvent with To state",
			event:    &TransitionEvent{To: &testState{Name: "target"}},
			expected: "{Name:To,Type:goat.AbstractState,Value:&{{0} target}}",
		},
		{
			name:     "testEventWithPointer",
			event:    &testEventWithPointer{ptr: &testStruct{value: 100}},
			expected: noFieldsMessage, // ptr field is skipped because it's a pointer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEventDetails(tt.event)
			if result != tt.expected {
				t.Errorf("getEventDetails(%T) = %s, expected %s", tt.event, result, tt.expected)
			}
		})
	}
}
