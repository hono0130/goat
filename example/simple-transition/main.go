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

	sm.StateMachine.New()
	sm.SetInitialState(stateA)

	sm.WithState(stateA,
		goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {
			this := sm.(*StateMachine)
			this.Mut = 1
			this.Goto(stateB, env)
		}),
	)

	sm.WithState(stateB,
		goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {
			this := sm.(*StateMachine)
			this.Mut = 2
			this.Goto(stateC, env)
		}),
	)

	sm.WithState(stateC,
		goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {
			this := sm.(*StateMachine)
			this.Mut = 3
		}),
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

	kripke.WriteAsDot(os.Stdout)
}
