package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
)

func TestSimpleTransition(t *testing.T) {
	opts := createSimpleTransitionModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.Summary.TotalWorlds != 8 {
		t.Fatalf("expected TotalWorlds=8, got %d", result.Summary.TotalWorlds)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}

	expected := goat.Violation{
		Rule: "Always mut<=1",
		Path: []goat.WorldSnapshot{
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:0}"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:1}"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "StateMachine", EventName: "exitEvent", Details: "no fields"},
					{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}"},
					{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:A}", Details: "{Name:Mut,Type:int,Value:1}"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}"},
					{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:B}", Details: "{Name:Mut,Type:int,Value:1}"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "StateMachine", State: "{Name:StateType,Type:main.StateType,Value:B}", Details: "{Name:Mut,Type:int,Value:2}"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "StateMachine", EventName: "exitEvent", Details: "no fields"},
					{TargetMachine: "StateMachine", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}"},
					{TargetMachine: "StateMachine", EventName: "entryEvent", Details: "no fields"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, result.Violations[0]); diff != "" {
		t.Errorf("violation mismatch (-want +got):\n%s", diff)
	}
}
