package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
)

func TestTemporalRuleViolationExample(t *testing.T) {
	opts := createTemporalRuleViolationModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.Summary.TotalWorlds != 11 {
		t.Fatalf("expected TotalWorlds=11, got %d", result.Summary.TotalWorlds)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}

	expected := goat.Violation{
		Rule: "whenever inPaid eventually inShipped",
		Path: []goat.WorldSnapshot{
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "FailingShipper", EventName: "entryEvent", Details: "no fields"},
					{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "Order", EventName: "exitEvent", Details: "no fields"},
					{TargetMachine: "Order", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} Paid}}"},
					{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Pending}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "Order", EventName: "transitionEvent", Details: "{Name:To,Type:goat.AbstractState,Value:&{{0} Paid}}"},
					{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "Order", EventName: "entryEvent", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{
					{TargetMachine: "FailingShipper", EventName: "eShipRequest", Details: "no fields"},
				},
			},
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{},
			},
		},
		Loop: []goat.WorldSnapshot{
			{
				StateMachines: []goat.StateMachineSnapshot{
					{Name: "FailingShipper", State: "no fields", Details: "no fields"},
					{Name: "Order", State: "{Name:StateType,Type:main.StateType,Value:Paid}", Details: "no fields"},
				},
				QueuedEvents: []goat.EventSnapshot{},
			},
		},
	}

	if diff := cmp.Diff(expected, result.Violations[0]); diff != "" {
		t.Errorf("violation mismatch (-want +got):\n%s", diff)
	}
}
