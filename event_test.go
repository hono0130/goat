package goat

import (
	"reflect"
	"testing"
)

type testEventWithPointer struct {
	Event[*testStateMachine, *testStateMachine]
	ptr *testStruct
}

type testStruct struct {
	value int
}

func TestCloneEvent(t *testing.T) {
	tests := []struct {
		name     string
		original AbstractEvent
		setup    func(AbstractEvent)
		validate func(*testing.T, AbstractEvent)
	}{
		{
			name:     "testEvent",
			original: &testEvent{Value: 42},
		},
		{
			name:     "entryEvent",
			original: &entryEvent{},
		},
		{
			name:     "exitEvent",
			original: &exitEvent{},
		},
		{
			name:     "haltEvent",
			original: &haltEvent{},
		},
		{
			name:     "transitionEvent",
			original: &transitionEvent{To: &testState{Name: "target"}},
		},
		{
			name:     "testEventWithPointer",
			original: &testEventWithPointer{ptr: &testStruct{value: 100}},
		},
		func() struct {
			name     string
			original AbstractEvent
			setup    func(AbstractEvent)
			validate func(*testing.T, AbstractEvent)
		} {
			sender := newTestStateMachine(newTestState("sender"))
			recipient := newTestStateMachine(newTestState("recipient"))
			return struct {
				name     string
				original AbstractEvent
				setup    func(AbstractEvent)
				validate func(*testing.T, AbstractEvent)
			}{
				name:     "preserves routing metadata",
				original: &testEvent{Value: 10},
				setup: func(ev AbstractEvent) {
					ev.(*testEvent).setRoutingInfo(sender, recipient)
				},
				validate: func(t *testing.T, cloned AbstractEvent) {
					clonedEvent := cloned.(*testEvent)
					if clonedEvent.Sender() != sender {
						t.Errorf("expected cloned sender %p, got %p", sender, clonedEvent.Sender())
					}
					if clonedEvent.Recipient() != recipient {
						t.Errorf("expected cloned recipient %p, got %p", recipient, clonedEvent.Recipient())
					}
				},
			}
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.original)
			}

			cloned := cloneEvent(tt.original)

			if reflect.ValueOf(tt.original).Pointer() == reflect.ValueOf(cloned).Pointer() {
				t.Errorf("Expected different pointer addresses, but got the same: %p", tt.original)
			}

			if reflect.TypeOf(tt.original) != reflect.TypeOf(cloned) {
				t.Errorf("Expected same type, but got different: %T vs %T", tt.original, cloned)
			}

			if !reflect.DeepEqual(tt.original, cloned) {
				t.Errorf("Expected original and cloned events to be equal, but they are not: %v vs %v", tt.original, cloned)
			}

			if tt.validate != nil {
				tt.validate(t, cloned)
			}
		})
	}
}

func TestEvent_Sender(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var ev Event[*testStateMachine, *testStateMachine]
		if ev.Sender() != nil {
			t.Errorf("expected default sender to be nil")
		}
	})

	t.Run("matching type", func(t *testing.T) {
		sender := newTestStateMachine(newTestState("sender"))
		recipient := newTestStateMachine(newTestState("recipient"))

		var ev Event[*testStateMachine, *testStateMachine]
		ev.setRoutingInfo(sender, recipient)

		if ev.Sender() != sender {
			t.Errorf("expected sender %p, got %p", sender, ev.Sender())
		}
	})

	t.Run("mismatched type clears typed sender", func(t *testing.T) {
		type otherTestStateMachine struct {
			StateMachine
		}
		sender := &otherTestStateMachine{}
		recipient := newTestStateMachine(newTestState("recipient"))

		var ev Event[*testStateMachine, *testStateMachine]
		ev.setRoutingInfo(sender, recipient)

		if ev.Sender() != nil {
			t.Errorf("expected typed sender to be nil when types do not match")
		}
	})
}

func TestEvent_Recipient(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var ev Event[*testStateMachine, *testStateMachine]
		if ev.Recipient() != nil {
			t.Errorf("expected default recipient to be nil")
		}
	})

	t.Run("matching type", func(t *testing.T) {
		sender := newTestStateMachine(newTestState("sender"))
		recipient := newTestStateMachine(newTestState("recipient"))

		var ev Event[*testStateMachine, *testStateMachine]
		ev.setRoutingInfo(sender, recipient)

		if ev.Recipient() != recipient {
			t.Errorf("expected recipient %p, got %p", recipient, ev.Recipient())
		}
	})

	t.Run("mismatched type clears typed recipient", func(t *testing.T) {
		sender := newTestStateMachine(newTestState("sender"))
		type otherTestStateMachine struct {
			StateMachine
		}
		recipient := &otherTestStateMachine{}

		var ev Event[*testStateMachine, *testStateMachine]
		ev.setRoutingInfo(sender, recipient)

		if ev.Recipient() != nil {
			t.Errorf("expected typed recipient to be nil when types do not match")
		}
	})
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
			name:     "same entryEvent types",
			event1:   &entryEvent{},
			event2:   &entryEvent{},
			expected: true,
		},
		{
			name:     "different types",
			event1:   &testEvent{},
			event2:   &entryEvent{},
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
			name:     "entryEvent",
			event:    &entryEvent{},
			expected: "entryEvent",
		},
		{
			name:     "exitEvent",
			event:    &exitEvent{},
			expected: "exitEvent",
		},
		{
			name:     "transitionEvent",
			event:    &transitionEvent{},
			expected: "transitionEvent",
		},
		{
			name:     "haltEvent",
			event:    &haltEvent{},
			expected: "haltEvent",
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
			name:     "entryEvent no fields",
			event:    &entryEvent{},
			expected: noFieldsMessage,
		},
		{
			name:     "transitionEvent with To state",
			event:    &transitionEvent{To: &testState{Name: "target"}},
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
