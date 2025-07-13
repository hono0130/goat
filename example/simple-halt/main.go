package main

import (
	"context"
	"fmt"
	"os"

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

func (sm *StateMachine) NewMachine() {
	var (
		stateA = &State{StateType: StateA}
		stateB = &State{StateType: StateB}
	)

	sm.StateMachine.New(stateA, stateB)
	sm.SetInitialState(stateA)

	goat.OnEntry(sm, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Halt(ctx, machine)
		// no longer reachable since the state machine is halted.
		goat.Goto(ctx, stateB)
	})

	goat.OnEntry(sm, stateB, func(ctx context.Context, machine *StateMachine) {
		fmt.Println("This should not be printed since the state machine is halted in state A.")
		goat.Goto(ctx, stateA)
	})
}

func main() {
	sm := &StateMachine{}
	sm.NewMachine()

	kripke, err := goat.KripkeModel(
		goat.WithStateMachines(sm),
	)
	if err != nil {
		panic(err)
	}
	if err := kripke.Solve(); err != nil {
		panic(err)
	}
	kripke.WriteAsLog(os.Stdout, "The state machine should halt in state A.")
}
