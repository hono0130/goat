package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSimpleHalt(t *testing.T) {
	opts := createSimpleHaltModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	fmt.Println(buf.String())

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// The simple-halt example should have 3 worlds showing the halt process
	expected := map[string]any{
		"worlds": []any{
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "no fields",
						"event_name":     "ExitEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "no fields",
						"event_name":     "HaltEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "no fields",
						"event_name":     "ExitEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
		},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreMapEntries(func(k, v any) bool {
			key, ok := k.(string)
			return ok && key == "id"
		}),
		// Ignore "summary" key since we only want to test worlds data
		cmpopts.IgnoreMapEntries(func(k, v any) bool {
			key, ok := k.(string)
			return ok && key == "summary"
		}),
	}

	if diff := cmp.Diff(expected, data, cmpOpts...); diff != "" {
		t.Errorf("JSON data mismatch (-expected +actual):\n%s", diff)
	}
}
