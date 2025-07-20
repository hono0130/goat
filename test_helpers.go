package goat

import (
	"testing"
)

type testStateMachine struct {
	StateMachine
}

type testState struct {
	State
	Name string
}

type testEvent struct {
	Event
	Value int
}

type testHandler struct{}

func (testHandler) handle(_ Environment, _ string, _ AbstractEvent) ([]localState, error) {
	return nil, nil
}

func newTestState(name string) *testState {
	return &testState{Name: name}
}

func newTestStateMachine(initialState AbstractState, states ...AbstractState) *testStateMachine {
	spec := NewStateMachineSpec(&testStateMachine{})
	spec.DefineStates(states...)
	spec.SetInitialState(initialState)
	sm := spec.NewInstance()
	return sm
}

func newTestEnvironment(machines ...*testStateMachine) Environment {
	env := Environment{
		machines: make(map[string]AbstractStateMachine),
		queue:    make(map[string][]AbstractEvent),
	}
	for _, sm := range machines {
		env.machines[sm.id()] = sm
	}
	return env
}

func newTestWorld(env Environment) world {
	return newWorld(env)
}

// Simplified test helpers - only keep essential factories

// Simple assertion functions for tests that need them
func assertEnvironmentEqual(t *testing.T, expected, actual Environment) {
	if len(expected.machines) != len(actual.machines) {
		t.Errorf("Expected %d machines, got %d", len(expected.machines), len(actual.machines))
	}
	if len(expected.queue) != len(actual.queue) {
		t.Errorf("Expected %d queues, got %d", len(expected.queue), len(actual.queue))
	}
}

func assertQueueEqual(t *testing.T, expected, actual map[string][]AbstractEvent) {
	if len(expected) != len(actual) {
		t.Errorf("Expected %d queues, got %d", len(expected), len(actual))
	}
}

func assertEventEqual(t *testing.T, expected, actual *testEvent) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		t.Errorf("Expected event %v, got %v", expected, actual)
		return
	}
	if expected.Value != actual.Value {
		t.Errorf("Expected value %d, got %d", expected.Value, actual.Value)
	}
}
