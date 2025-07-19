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

func TestSimpleTransition(t *testing.T) {
	_, opts := createSimpleTransitionModel()

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
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:0}",
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
						"details": "{Name:Mut,Type:int,Value:1}",
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
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} B}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:1}",
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
						"details": "{Name:Mut,Type:int,Value:1}",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": true,
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
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:2}",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": true,
				"queued_events": []interface{}{
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]interface{}{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:2}",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": true,
				"queued_events": []interface{}{
					map[string]interface{}{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:2}",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:C}",
					},
				},
			},
			map[string]interface{}{
				"invariant_violation": true,
				"queued_events":       []interface{}{},
				"state_machines": []interface{}{
					map[string]interface{}{
						"details": "{Name:Mut,Type:int,Value:3}",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:C}",
					},
				},
			},
		},
	}

	cmpOpts := cmp.Options{
		// Ignore "id" keys in maps (though they should be gone now)
		cmpopts.IgnoreMapEntries(func(k, v interface{}) bool {
			key, ok := k.(string)
			return ok && key == "id"
		}),
	}

	if diff := cmp.Diff(expected, data, cmpOpts...); diff != "" {
		t.Errorf("JSON data mismatch (-expected +actual):\n%s", diff)
	}
}