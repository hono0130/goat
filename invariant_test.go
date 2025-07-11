package goat

import (
	"testing"
)

func TestRef_Statemachine(t *testing.T) {

	tests := []struct {
		name         string
		initiate     func() (world, *Ref, AbstractStateMachine)
		expectExists bool
	}{
		{
			name: "ToRef creates correct reference",
			initiate: func() (world, *Ref, AbstractStateMachine) {
				sm := NewTestStateMachine("initial")
				return NewTestWorld(NewTestEnvironment(sm)), ToRef(sm), sm
			},
			expectExists: true,
		},
		{
			name: "Non-existent state machine",
			initiate: func() (world, *Ref, AbstractStateMachine) {
				sm := NewTestStateMachine("initial")
				nonExistentRef := &Ref{id: "non-existent"}
				return NewTestWorld(NewTestEnvironment(sm)), nonExistentRef, sm
			},
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			world, ref, sm := tt.initiate()
			retrievedSm, exists := ref.statemachine(world)
			if exists != tt.expectExists {
				t.Errorf("Expected exists to be %v, but got %v", tt.expectExists, exists)
			}
			if !exists {
				return 
			}

			if retrievedSm.id() != sm.id() {
				t.Errorf("Expected retrieved state machine id to be %s, but got %s", sm.id(), retrievedSm.id())
			}
		})
	}
}

func TestRef_Invariant(t *testing.T) {
	tests := []struct {
		name        string
		initiate    func() (world, *Ref)
		invariantFn func(sm AbstractStateMachine) bool
		wantResult  bool
	}{
		{
			name: "always true invariant",
			initiate: func() (world, *Ref) {
				sm := NewTestStateMachine("initial")
				return NewTestWorld(NewTestEnvironment(sm)), ToRef(sm)
			},
			invariantFn: func(sm AbstractStateMachine) bool {
				return true
			},
			wantResult: true,
		},
		{
			name: "always false invariant",
			initiate: func() (world, *Ref) {
				sm := NewTestStateMachine("initial")
				return NewTestWorld(NewTestEnvironment(sm)), ToRef(sm)
			},
			invariantFn: func(sm AbstractStateMachine) bool {
				return false
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ref := tt.initiate()
			invariant := ref.Invariant(tt.invariantFn)
			
			got := invariant.Evaluate(w)
			if got != tt.wantResult {
				t.Errorf("Evaluate() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
