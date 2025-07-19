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
	_, opts := createSimpleHaltModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	fmt.Println(buf.String())

	var data map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// The simple-halt example should have 3 worlds showing the halt process
	expected := map[string]interface{}{
		"worlds": []interface{}{
			map[string]interface{}{
				"invariant_violation": false,
				"queued_events": []interface{}{
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "ExitEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "HaltEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": false,
				"queued_events": []interface{}{
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "ExitEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": false,
				"queued_events": []interface{}{
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
		},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreMapEntries(func(k, v interface{}) bool {
			key, ok := k.(string)
			return ok && key == "id"
		}),
	}

	if diff := cmp.Diff(expected, data, cmpOpts...); diff != "" {
		t.Errorf("JSON data mismatch (-expected +actual):\n%s", diff)
	}
}