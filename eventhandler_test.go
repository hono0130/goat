package goat

import (
	"context"
	"testing"
)

func TestOnEvent(t *testing.T) {
	t.Run("registers handler builder for specified state and event", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{name: "test"}
		event := &testEvent{value: 1}

		OnEvent(spec, state, event, func(ctx context.Context, e *testEvent, sm *testStateMachine) {})

		// Validate that handler builder was registered for the correct state
		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, event) {
			t.Error("Handler builder should be registered for the specified event")
		}

	})
}

func TestOnEntry(t *testing.T) {
	t.Run("registers entry handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{name: "entry_test"}

		OnEntry(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		// Validate that handler builder was registered for the correct state
		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &EntryEvent{}) {
			t.Error("Handler builder should be registered for EntryEvent")
		}
	})
}

func TestOnExit(t *testing.T) {
	t.Run("registers exit handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{name: "exit_test"}

		OnExit(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		// Validate that handler builder was registered for the correct state
		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &ExitEvent{}) {
			t.Error("Handler builder should be registered for ExitEvent")
		}
	})
}

func TestOnTransition(t *testing.T) {
	t.Run("registers transition handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{name: "transition_test"}

		OnTransition(spec, state, func(ctx context.Context, toState AbstractState, sm *testStateMachine) {})

		// Validate that handler builder was registered for the correct state
		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &TransitionEvent{}) {
			t.Error("Handler builder should be registered for TransitionEvent")
		}
	})
}

func TestOnHalt(t *testing.T) {
	t.Run("registers halt handler builder for specified state", func(t *testing.T) {
		spec := &StateMachineSpec[*testStateMachine]{
			prototype:       &testStateMachine{},
			handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
		}
		state := &testState{name: "halt_test"}

		OnHalt(spec, state, func(ctx context.Context, sm *testStateMachine) {})

		// Validate that handler builder was registered for the correct state
		builders, exists := spec.handlerBuilders[state]
		if !exists {
			t.Error("Handler builder should be registered for the specified state")
		}
		if len(builders) != 1 {
			t.Error("Exactly one handler builder should be registered")
		}

		builderInfo := builders[0]
		if !sameEvent(builderInfo.event, &HaltEvent{}) {
			t.Error("Handler builder should be registered for HaltEvent")
		}
	})
}

func TestDefaultOnTransitionHandler_handle(t *testing.T) {
	t.Run("returns single local state with updated state machine", func(t *testing.T) {
		handler := &defaultOnTransitionHandler{}
		initialState := newTestState("initial")
		targetState := newTestState("target")
		sm := newTestStateMachine(initialState, targetState)
		env := newTestEnvironment(sm)
		event := &TransitionEvent{To: targetState}

		states, err := handler.handle(env, sm.id(), event)
		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if len(states) != 1 {
			t.Errorf("Expected 1 state, got %d", len(states))
		}
		newSM := states[0].env.machines[sm.id()]
		if !sameState(newSM.currentState(), targetState) {
			t.Error("State should be changed to target state")
		}
	})
}

func TestDefaultOnHaltHandler_handle(t *testing.T) {
	t.Run("returns single local state with halted state machine", func(t *testing.T) {
		handler := &defaultOnHaltHandler{}
		initialState := newTestState("initial")
		sm := newTestStateMachine(initialState)
		env := newTestEnvironment(sm)
		event := &HaltEvent{}

		states, err := handler.handle(env, sm.id(), event)

		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if len(states) != 1 {
			t.Errorf("Expected 1 state, got %d", len(states))
		}
		newSM := states[0].env.machines[sm.id()]
		innerSM := getInnerStateMachine(newSM)
		if !innerSM.halted {
			t.Error("State machine should be halted")
		}
	})
}

func TestEventHandlers_handle(t *testing.T) {
	t.Run("returns nil for non-matching event types", func(t *testing.T) {
		handlers := &eventHandlers{
			fs:    []eventHandler{},
			event: &testEvent{value: 1},
		}
		env := Environment{
			machines: make(map[string]AbstractStateMachine),
			queue:    make(map[string][]AbstractEvent),
		}
		wrongEvent := &EntryEvent{}

		states, err := handlers.handle(env, "test", wrongEvent)

		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if states != nil {
			t.Error("Should return nil for non-matching event types")
		}
	})

	t.Run("processes matching event types and returns local states", func(t *testing.T) {
		called := false
		testHandler := func(event AbstractEvent, env *Environment) {
			called = true
		}
		
		handlers := &eventHandlers{
			fs:    []eventHandler{testHandler},
			event: &testEvent{value: 1},
		}
		env := Environment{
			machines: make(map[string]AbstractStateMachine),
			queue:    make(map[string][]AbstractEvent),
		}
		matchingEvent := &testEvent{value: 1}

		states, err := handlers.handle(env, "test", matchingEvent)

		if err != nil {
			t.Errorf("handle() error = %v", err)
		}
		if len(states) != 1 {
			t.Errorf("Expected 1 state, got %d", len(states))
		}
		if !called {
			t.Error("Event handler should have been called")
		}
	})
}
