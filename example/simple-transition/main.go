package main

import (
	"context"
	"os"

	"github.com/goatx/goat"
)

type (
	StateType string
)

const (
	StateA StateType = "A"
	StateB StateType = "B"
	StateC StateType = "C"
)

type (
	State struct {
		goat.State
		StateType StateType
	}
)

type StateMachine struct {
	goat.StateMachine
	Mut int
}

func (sm *StateMachine) NewMachine() {
	var (
		stateA = &State{StateType: StateA}
		stateB = &State{StateType: StateB}
		stateC = &State{StateType: StateC}
	)

	sm.StateMachine.New(stateA, stateB, stateC)
	sm.SetInitialState(stateA)

	goat.OnEntry(sm, stateA, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 1
		goat.Goto(ctx, stateB)
	})

	goat.OnEntry(sm, stateB, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 2
		goat.Goto(ctx, stateC)
	})

	goat.OnEntry(sm, stateC, func(ctx context.Context, machine *StateMachine) {
		machine.Mut = 3
	})
}

func main() {
	sm := &StateMachine{}
	sm.NewMachine()

	ref := goat.ToRef(sm)
	kripke, err := goat.KripkeModel(
		goat.WithStateMachines(sm),
		goat.WithInvariants(
			ref.Invariant(func(sm goat.AbstractStateMachine) bool {
				return sm.(*StateMachine).Mut <= 1
			}),
		),
	)
	if err != nil {
		panic(err)
	}

	if err := kripke.Solve(); err != nil {
		panic(err)
	}

	kripke.WriteAsLog(os.Stdout, "Mut <= 1")
}
