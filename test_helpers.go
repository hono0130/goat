package goat

import "testing"

type TestStateMachine struct {
	StateMachine
}

type TestState struct {
	State
	name string
}

type TestEvent struct {
	Event
	value int
}

type TestHandler struct{}

func (TestHandler) handle(_ Environment, _ string, _ AbstractEvent) ([]localState, error) {
	return nil, nil
}

func NewTestState(name string) *TestState {
	return &TestState{name: name}
}

func NewTestStateMachine(stateName string) *TestStateMachine {
	sm := &TestStateMachine{}
	sm.New()
	sm.SetInitialState(&TestState{name: stateName})
	return sm
}

func NewTestEnvironment(machines ...*TestStateMachine) Environment {
	env := Environment{
		machines: make(map[string]AbstractStateMachine),
		queue:    make(map[string][]AbstractEvent),
	}
	for _, sm := range machines {
		env.machines[sm.id()] = sm
	}
	return env
}

func NewTestWorld(env Environment) world {
	return newWorld(env)
}

func AssertEventEqual(t *testing.T, expected, actual *TestEvent) {
	t.Helper()
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		t.Errorf("Expected event %v, but got %v", expected, actual)
		return
	}
	if expected.value != actual.value {
		t.Errorf("Expected TestEvent with value %d, but got %d", expected.value, actual.value)
	}
}

func AssertQueueEqual(t *testing.T, expected, actual map[string][]AbstractEvent) {
	t.Helper()

	// Check length
	if len(expected) != len(actual) {
		t.Errorf("Expected queue with %d state machines, but got %d", len(expected), len(actual))
		return
	}

	// Check each state machine's queue
	for smID, expectedEvents := range expected {
		actualEvents, ok := actual[smID]
		if !ok {
			t.Errorf("Queue for machine %s not found in actual queue", smID)
			continue
		}

		if len(expectedEvents) != len(actualEvents) {
			t.Errorf("Expected %d events for machine %s, but got %d", len(expectedEvents), smID, len(actualEvents))
			continue
		}

		// Compare each event
		for i, expectedEvent := range expectedEvents {
			actualEvent := actualEvents[i]
			// Compare event types
			if getEventName(expectedEvent) != getEventName(actualEvent) {
				t.Errorf("Expected event type %s at index %d, but got %s", getEventName(expectedEvent), i, getEventName(actualEvent))
				continue
			}
			// For TestEvent, compare values
			if testExpected, ok := expectedEvent.(*TestEvent); ok {
				testActual := actualEvent.(*TestEvent)
				if testExpected.value != testActual.value {
					t.Errorf("Expected TestEvent with value %d, but got %d", testExpected.value, testActual.value)
				}
			}
			// For TransitionEvent, compare target states
			if transExpected, ok := expectedEvent.(*TransitionEvent); ok {
				transActual := actualEvent.(*TransitionEvent)
				if !sameState(transExpected.To, transActual.To) {
					t.Errorf("Expected TransitionEvent to state %v, but got %v", transExpected.To, transActual.To)
				}
			}
		}
	}

	// Check for unexpected state machines
	for smID := range actual {
		if _, ok := expected[smID]; !ok {
			t.Errorf("Unexpected queue for machine %s found in actual queue", smID)
		}
	}
}

func AssertStateMachinesEqual(t *testing.T, expected, actual map[string]AbstractStateMachine) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("Expected %d machines, but got %d", len(expected), len(actual))
		return
	}

	// Check each machine exists (not comparing instances, just existence)
	for smID := range expected {
		if _, ok := actual[smID]; !ok {
			t.Errorf("Machine with id %s not found in actual machines", smID)
		}
	}

	for smID := range actual {
		if _, ok := expected[smID]; !ok {
			t.Errorf("Unexpected machine with id %s found in actual machines", smID)
		}
	}
}

func AssertEnvironmentEqual(t *testing.T, expected, actual Environment) {
	t.Helper()

	// Check machines
	AssertStateMachinesEqual(t, expected.machines, actual.machines)

	// Check queues
	AssertQueueEqual(t, expected.queue, actual.queue)
}
