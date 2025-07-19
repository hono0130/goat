package main

import (
	"context"

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

func createSimpleNonDeterministicModel() (*StateMachine, []goat.Option) {
	// === StateMachine Spec ===
	spec := goat.NewStateMachineSpec(&StateMachine{})

	// StateMachineが取る状態を定義
	stateA := &State{StateType: StateA}
	stateB := &State{StateType: StateB}
	stateC := &State{StateType: StateC}

	// StateMachineを初期化
	spec.DefineStates(stateA, stateB, stateC).SetInitialState(stateA)

	// 状態Aにおける処理
	// 状態Aに遷移した際に以下の処理を非決定的に実行
	// 1. 状態Bに遷移
	// 2. 状態Cに遷移
	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateB)
	})
	goat.OnEntry(spec, stateA, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateC)
	})

	// 状態Bにおける処理
	// 状態Bに遷移した際に以下の処理を非決定的に実行
	// 1. 状態Cに遷移
	// 2. 状態Aに遷移
	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateC)
	})
	goat.OnEntry(spec, stateB, func(ctx context.Context, machine *StateMachine) {
		goat.Goto(ctx, stateA)
	})

	// === Create Instance ===
	sm := spec.NewInstance()

	opts := []goat.Option{
		goat.WithStateMachines(sm),
	}

	return sm, opts
}

func main() {
	_, opts := createSimpleNonDeterministicModel()
	
	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
