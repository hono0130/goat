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
}

func (sm *StateMachine) NewMachine() {
	// StateMachineが取る状態を定義
	var (
		stateA = &State{StateType: StateA}
		stateB = &State{StateType: StateB}
		stateC = &State{StateType: StateC}
	)

	// StateMachineを初期化
	sm.StateMachine.New(stateA, stateB, stateC)
	// 初期状態を設定
	sm.SetInitialState(stateA)

	// 状態Aにおける処理
	// 状態Aに遷移した際に以下の処理を非決定的に実行
	// 1. 状態Bに遷移
	// 2. 状態Cに遷移
	sm.OnEntry(stateA,
			func(env *goat.Environment) {
				sm.Goto(stateB, env)
			},
			func(env *goat.Environment) {
				sm.Goto(stateC, env)
			},
	)

	// 状態Bにおける処理
	// 状態Bに遷移した際に以下の処理を非決定的に実行
	// 1. 状態Cに遷移
	// 2. 状態Aに遷移
	sm.OnEntry(stateB,
			func(env *goat.Environment) {
				sm.Goto(stateC, env)
			},
			func(env *goat.Environment) {
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
	kripke.WriteAsLog(os.Stdout, "The state machine should transition to state B or state C.")
}
