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

func TestSimpleNonDeterministic(t *testing.T) {
	opts := createSimpleNonDeterministicModel()

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

	expected := map[string]any{
		"worlds": []any{
			// World 1: A state, EntryEvent queued
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
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			// World 2: A state, EntryEvent + ExitEvent + TransitionEvent(B) queued
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
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			// World 3: A state, EntryEvent + ExitEvent + TransitionEvent(C) queued
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
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			// World 4: A state, EntryEvent + TransitionEvent(B) queued
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
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
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			// World 5: A state, EntryEvent + TransitionEvent(C) queued
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:A}",
					},
				},
			},
			// World 6: B state, EntryEvent queued
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
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			// World 7: B state, EntryEvent + ExitEvent + TransitionEvent(A) queued
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
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} A}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			// World 8: B state, EntryEvent + ExitEvent + TransitionEvent(C) queued
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
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			// World 9: B state, EntryEvent + TransitionEvent(A) queued
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} A}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			// World 10: B state, EntryEvent + TransitionEvent(C) queued
			map[string]any{
				"invariant_violation": false,
				"queued_events": []any{
					map[string]any{
						"details":        "no fields",
						"event_name":     "EntryEvent",
						"target_machine": "StateMachine",
					},
					map[string]any{
						"details":        "{Name:To,Type:goat.AbstractState,Value:&{{0} C}}",
						"event_name":     "TransitionEvent",
						"target_machine": "StateMachine",
					},
				},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:B}",
					},
				},
			},
			// World 11: C state, no events queued
			map[string]any{
				"invariant_violation": false,
				"queued_events":       []any{},
				"state_machines": []any{
					map[string]any{
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:C}",
					},
				},
			},
			// World 12: C state, EntryEvent queued
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
						"id":      "StateMachine",
						"details": "no fields",
						"name":    "StateMachine",
						"state":   "{Name:StateType,Type:main.StateType,Value:C}",
					},
				},
			},
		},
	}

	cmpOpts := cmp.Options{
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
