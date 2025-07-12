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

	sm.StateMachine.New(stateA, stateB)
	sm.SetInitialState(stateA)

	sm.OnEntry(stateA,
			func(env *goat.Environment) {
				sm.Halt(sm, env)
				// no longer reachable since the state machine is halted.
				sm.Goto(stateB, env)
			},
	)

	sm.OnEntry(stateB,
			func(env *goat.Environment) {
				fmt.Println("This should not be printed since the state machine is halted in state A.")
				sm.Goto(stateA, env)
			},
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
	kripke.WriteAsLog(os.Stdout, "The state machine should halt in state A.")
}
