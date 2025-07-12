package main

import (
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

	sm.OnEntry(stateA,
		func(env *goat.Environment) {
			sm.Mut = 1
			sm.Goto(stateB, env)
		},
	)

	sm.OnEntry(stateB,
		func(env *goat.Environment) {
			sm.Mut = 2
			sm.Goto(stateC, env)
		},
	)

	sm.OnEntry(stateC,
		func(env *goat.Environment) {
			sm.Mut = 3
		},
	)
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
