package main

import (
	"context"
	"fmt"

	"github.com/goatx/goat"
)

type (
	StateType string
)

const (
	StateA StateType = "A"
	StateB StateType = "B"
)

type (
	State struct {
		goat.State
		StateType StateType
	}
)

type StateMachine struct {
	goat.StateMachine
}

func main() {
	// === StateMachine Spec ===
	spec := goat.NewStateMachineSpec(&StateMachine{})
	stateA := &State{StateType: StateA}
	stateB := &State{StateType: StateB}

	spec.DefineStates(stateA, stateB).SetInitialState(stateA)

	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Halt(ctx, machine)
		// no longer reachable since the state machine is halted.
		goat.Goto(ctx, stateB)
	})

	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		fmt.Println("This should not be printed since the state machine is halted in state A.")
		goat.Goto(ctx, stateA)
	})

	// === Create Instance ===
	sm := spec.NewInstance()

	err := goat.Test(
		goat.WithStateMachines(sm),
	)
	if err != nil {
		panic(err)
	}
}
