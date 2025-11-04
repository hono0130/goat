package goat

import "context"

type testStateMachine struct {
	StateMachine
}

type testState struct {
	State
	Name string
}

type testEvent struct {
	Event[*testStateMachine, *testStateMachine]
	Value int
}

const testStateMachineID = "testStateMachine"

func newTestState(name string) *testState {
	return &testState{Name: name}
}

func newTestStateMachine(initialState AbstractState, states ...AbstractState) *testStateMachine {
	spec := NewStateMachineSpec(&testStateMachine{})
	allStates := append([]AbstractState{initialState}, states...)
	spec.DefineStates(allStates...)
	spec.SetInitialState(initialState)
	sm, err := spec.NewInstance()
	if err != nil {
		panic(err.Error()) // Test helper can panic for simplicity
	}
	return sm
}

func newTestEnvironment(machines ...*testStateMachine) environment {
	env := environment{
		machines: make(map[string]AbstractStateMachine),
		queue:    make(map[string][]AbstractEvent),
	}
	for _, sm := range machines {
		env.machines[sm.id()] = sm
	}
	return env
}

func newTestWorld(env environment) world {
	return newWorld(env)
}

// NewTestContext creates a context with a minimal environment for testing.
// This is useful for executing handlers outside of the normal model checking flow.
func NewTestContext(sm AbstractStateMachine) context.Context {
	env := environment{
		machines: make(map[string]AbstractStateMachine),
		queue:    make(map[string][]AbstractEvent),
	}
	if sm != nil {
		env.machines[sm.id()] = sm
	}
	return withEnvAndSM(&env, sm)
}
