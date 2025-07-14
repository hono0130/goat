package goat

import (
	"testing"
)

func TestInvariantFor(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*TestStateMachine, world)
		checkFunc  func(*TestStateMachine) bool
		wantResult bool
	}{
		{
			name: "always true invariant",
			setup: func() (*TestStateMachine, world) {
				sm := NewTestStateMachine("initial")
				return sm, NewTestWorld(NewTestEnvironment(sm))
			},
			checkFunc: func(sm *TestStateMachine) bool {
				return true
			},
			wantResult: true,
		},
		{
			name: "always false invariant",
			setup: func() (*TestStateMachine, world) {
				sm := NewTestStateMachine("initial")
				return sm, NewTestWorld(NewTestEnvironment(sm))
			},
			checkFunc: func(sm *TestStateMachine) bool {
				return false
			},
			wantResult: false,
		},
		{
			name: "check specific state",
			setup: func() (*TestStateMachine, world) {
				sm := NewTestStateMachine("initial")
				return sm, NewTestWorld(NewTestEnvironment(sm))
			},
			checkFunc: func(sm *TestStateMachine) bool {
				// Access the embedded StateMachine
				currentState := sm.currentState().(*TestState)
				return currentState.name == "initial"
			},
			wantResult: true,
		},
		{
			name: "non-existent state machine",
			setup: func() (*TestStateMachine, world) {
				sm := NewTestStateMachine("initial")
				// Create world without this state machine
				emptyEnv := Environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
				return sm, NewTestWorld(emptyEnv)
			},
			checkFunc: func(sm *TestStateMachine) bool {
				return true
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, w := tt.setup()
			invariant := NewInvariant(sm, tt.checkFunc)
			
			got := invariant.Evaluate(w)
			if got != tt.wantResult {
				t.Errorf("Evaluate() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestBoolInvariant(t *testing.T) {
	tests := []struct {
		name       string
		value      bool
		wantResult bool
	}{
		{
			name:       "true invariant",
			value:      true,
			wantResult: true,
		},
		{
			name:       "false invariant",
			value:      false,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewTestWorld(NewTestEnvironment())
			invariant := BoolInvariant(tt.value)
			
			got := invariant.Evaluate(w)
			if got != tt.wantResult {
				t.Errorf("Evaluate() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}