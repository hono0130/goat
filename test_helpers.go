package goat

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
