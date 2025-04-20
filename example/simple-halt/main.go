package main

import (
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

	sm.StateMachine.New()
	sm.SetInitialState(stateA)

	sm.WithState(stateA,
		goat.WithOnEntry(
			func(sm goat.AbstractStateMachine, env *goat.Environment) {
				this := sm.(*StateMachine)
				this.Halt(this, env)
				// no longer reachable since the state machine is halted.
				this.Goto(stateB, env)
			},
		),
	)

	sm.WithState(stateB,
		goat.WithOnEntry(
			func(sm goat.AbstractStateMachine, env *goat.Environment) {
				fmt.Println("This should not be printed since the state machine is halted in state A.")
				this := sm.(*StateMachine)
				this.Goto(stateA, env)
			},
		),
	)
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
	kripke.WriteAsDot(os.Stdout)
}
