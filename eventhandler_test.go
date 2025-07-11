package goat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnEntryHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *onEntryHandler
		environment		 Environment
		initiate        func() (Environment, AbstractStateMachine)
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantError       bool
	}{
		{
			name: "handles entry event",
			handler: &onEntryHandler{
				fs: []OnEntryFunc{
					func(sm AbstractStateMachine, env *Environment) {
						// Add an event to the queue
						env.enqueueEvent(sm, &TestEvent{value: 42})
					},
				},
			},

			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedEnv.queue[sm.id()] = []AbstractEvent{&TestEvent{value: 42}}
				return []localState{{env: expectedEnv}}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), &EntryEvent{})
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}
		})
	}
}

func TestOnExitHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *onExitHandler
		initiate        func() (Environment, AbstractStateMachine)
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantError       bool
	}{
		{
			name: "handles exit event",
			handler: &onExitHandler{
				fs: []OnExitFunc{
					func(sm AbstractStateMachine, env *Environment) {
						// Add a transition event to the queue
						env.enqueueEvent(sm, &TransitionEvent{To: &TestState{name: "next"}})
					},
				},
			},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedEnv.queue[sm.id()] = []AbstractEvent{&TransitionEvent{To: &TestState{name: "next"}}}
				return []localState{{env: expectedEnv}}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), &ExitEvent{})
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}
		})
	}
}

func TestOnEventHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *onEventHandler
		initiate        func() (Environment, AbstractStateMachine)
		event           AbstractEvent
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantError       bool
	}{
		{
			name: "handles matching event type",
			handler: &onEventHandler{
				event: &TestEvent{value: 42},
				fs: []OnEventFunc{
					func(sm AbstractStateMachine, event AbstractEvent, env *Environment) {
						// Add another event to the queue
						env.enqueueEvent(sm, &EntryEvent{})
					},
				},
			},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			event: &TestEvent{value: 99}, // Same type, different value
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedEnv.queue[sm.id()] = []AbstractEvent{&EntryEvent{}}
				return []localState{{env: expectedEnv}}
			},
			wantError: false,
		},
		{
			name: "ignores different event type",
			handler: &onEventHandler{
				event: &TestEvent{value: 42},
				fs: []OnEventFunc{
					func(sm AbstractStateMachine, event AbstractEvent, env *Environment) {},
				},
			},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			event: &EntryEvent{}, // Different event type
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				return []localState{}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), tt.event)
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}
		})
	}
}

func TestOnTransitionHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *onTransitionHandler
		initiate        func() (Environment, AbstractStateMachine)
		event           AbstractEvent
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantNewState    string
		wantError       bool
	}{
		{
			name: "handles transition event and changes state",
			handler: &onTransitionHandler{
				fs: []OnTransitionFunc{
					func(sm AbstractStateMachine, toState AbstractState, env *Environment) {
						// Add an exit event after transition
						env.enqueueEvent(sm, &ExitEvent{})
					},
				},
			},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("from")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			event: &TransitionEvent{To: &TestState{name: "to"}},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedEnv.queue[sm.id()] = []AbstractEvent{&ExitEvent{}}
				// Update state in the expected environment
				expectedSm := expectedEnv.machines[sm.id()]
				expectedSm.setCurrentState(&TestState{name: "to"})
				return []localState{{env: expectedEnv}}
			},
			wantNewState: "to",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), tt.event)
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}

			// Check that state was changed in the returned local state
			if len(got) > 0 {
				updatedSm := got[0].env.machines[sm.id()]
				assert.Equal(t, tt.wantNewState, updatedSm.currentState().(*TestState).name, "state transition failed")
			}
		})
	}
}

func TestDefaultOnTransitionHandler_Apply(t *testing.T) {
	tests := []struct {
		name     string
		initiate func() (StateMachine, AbstractState)
		want     func(StateMachine, AbstractState) StateMachine
	}{
		{
			name: "applies handler when no transition handler exists",
			initiate: func() (StateMachine, AbstractState) {
				sm := NewTestStateMachine("test")
				state := NewTestState("test")
				return sm.StateMachine, state
			},
			want: func(sm StateMachine, state AbstractState) StateMachine {
				sm.EventHandlers[state] = append(sm.EventHandlers[state], handlerInfo{
					event:   &TransitionEvent{},
					handler: &defaultOnTransitionHandler{},
				})
				return sm
			},
		},
		{
			name: "does not apply when transition handler already exists",
			initiate: func() (StateMachine, AbstractState) {
				sm := NewTestStateMachine("test")
				state := NewTestState("test")
				sm.EventHandlers[state] = append(sm.EventHandlers[state], handlerInfo{
					event:   &TransitionEvent{},
					handler: TestHandler{},
				})
				return sm.StateMachine, state
			},
			want: func(sm StateMachine, state AbstractState) StateMachine {
				// No new handler should be added
				return sm
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, state := tt.initiate()
			want := tt.want(sm, state)

			handler := &defaultOnTransitionHandler{}
			handler.apply(&sm, state)

			assert.Equal(t, want.EventHandlers[state], sm.EventHandlers[state], "event handlers should match after applying default handler")
		})
	}
}

func TestDefaultOnTransitionHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *defaultOnTransitionHandler
		initiate        func() (Environment, AbstractStateMachine)
		event           AbstractEvent
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantNewState    string
		wantError       bool
	}{
		{
			name:    "transitions to new state",
			handler: &defaultOnTransitionHandler{},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("from")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			event: &TransitionEvent{To: &TestState{name: "to"}},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedSm := expectedEnv.machines[sm.id()]
				expectedSm.setCurrentState(&TestState{name: "to"})
				return []localState{{env: expectedEnv}}
			},
			wantNewState: "to",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), tt.event)
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}

			// Check that state was changed
			if len(got) > 0 {
				updatedSm := got[0].env.machines[sm.id()]
				assert.Equal(t, tt.wantNewState, updatedSm.currentState().(*TestState).name, "state transition failed")
			}
		})
	}
}

func TestOnHaltHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *onHaltHandler
		initiate        func() (Environment, AbstractStateMachine)
		event           AbstractEvent
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantHalted      bool
		wantError       bool
	}{
		{
			name: "handles halt event",
			handler: &onHaltHandler{
				fs: []OnHaltFunc{
					func(sm AbstractStateMachine, env *Environment) {
						// Clear the queue on halt
						env.queue[sm.id()] = []AbstractEvent{}
					},
				},
			},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				return env, sm
			},
			event: &HaltEvent{},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedSm := expectedEnv.machines[sm.id()]
				innerSm := getInnerStateMachine(expectedSm)
				innerSm.halted = true                          // Should be halted
				expectedEnv.queue[sm.id()] = []AbstractEvent{} // Queue should be empty
				return []localState{{env: expectedEnv}}
			},
			wantHalted: true,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), tt.event)
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}

			// Check that state machine was halted
			if len(got) > 0 && tt.wantHalted {
				updatedSm := got[0].env.machines[sm.id()]
				innerSm := getInnerStateMachine(updatedSm)
				assert.True(t, innerSm.halted, "state machine should be halted")
			}
		})
	}
}

func TestDefaultOnHaltHandler_Apply(t *testing.T) {
	tests := []struct {
		name     string
		initiate func() (StateMachine, AbstractState)
		want     func(StateMachine, AbstractState) StateMachine
	}{
		{
			name: "applies handler when no halt handler exists",
			initiate: func() (StateMachine, AbstractState) {
				sm := NewTestStateMachine("test")
				state := NewTestState("test")
				return sm.StateMachine, state
			},
			want: func(sm StateMachine, state AbstractState) StateMachine {
				sm.EventHandlers[state] = append(sm.EventHandlers[state], handlerInfo{
					event:   &HaltEvent{},
					handler: &defaultOnHaltHandler{},
				})
				return sm
			},
		},
		{
			name: "does not apply when halt handler already exists",
			initiate: func() (StateMachine, AbstractState) {
				sm := NewTestStateMachine("test")
				state := NewTestState("test")
				sm.EventHandlers[state] = append(sm.EventHandlers[state], handlerInfo{
					event:   &HaltEvent{},
					handler: &onHaltHandler{fs: []OnHaltFunc{func(sm AbstractStateMachine, env *Environment) {}}},
				})
				return sm.StateMachine, state
			},
			want: func(sm StateMachine, state AbstractState) StateMachine {
				// No new handler should be added
				return sm
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, state := tt.initiate()
			want := tt.want(sm, state)

			handler := &defaultOnHaltHandler{}
			handler.apply(&sm, state)

			assert.Equal(t, want.EventHandlers[state], sm.EventHandlers[state], "event handlers should match after applying default handler")
		})
	}
}

func TestDefaultOnHaltHandler_Handle(t *testing.T) {
	tests := []struct {
		name            string
		handler         *defaultOnHaltHandler
		initiate        func() (Environment, AbstractStateMachine)
		event           AbstractEvent
		wantLocalStates func(env Environment, sm AbstractStateMachine) []localState
		wantHalted      bool
		wantError       bool
	}{
		{
			name:    "halts state machine",
			handler: &defaultOnHaltHandler{},
			initiate: func() (Environment, AbstractStateMachine) {
				sm := NewTestStateMachine("test")
				env := NewTestEnvironment(sm)
				env.queue[sm.id()] = []AbstractEvent{&TestEvent{value: 100}} // Has some events in queue
				return env, sm
			},
			event: &HaltEvent{},
			wantLocalStates: func(env Environment, sm AbstractStateMachine) []localState {
				expectedEnv := env.clone()
				expectedSm := expectedEnv.machines[sm.id()]
				innerSm := getInnerStateMachine(expectedSm)
				innerSm.halted = true // Should be halted
				// Queue remains unchanged from initiate
				return []localState{{env: expectedEnv}}
			},
			wantHalted: true,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, sm := tt.initiate()
			got, err := tt.handler.handle(env, sm.id(), tt.event)
			if tt.wantError {
				assert.Error(t, err, "expected error but got none")
				return
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			want := tt.wantLocalStates(env, sm)
			for i, ls := range got {
				AssertQueueEqual(t, ls.env.queue, want[i].env.queue)
				AssertStateMachinesEqual(t, ls.env.machines, want[i].env.machines)
			}

			// Check that state machine was halted
			if len(got) > 0 && tt.wantHalted {
				updatedSm := got[0].env.machines[sm.id()]
				innerSm := getInnerStateMachine(updatedSm)
				assert.True(t, innerSm.halted, "state machine should be halted")
			}
		})
	}
}
