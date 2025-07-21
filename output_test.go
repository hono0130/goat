package goat

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKripke_WriteAsDot(t *testing.T) {
	tests := []struct {
		name  string
		setup func() kripke
		want  string
	}{
		{
			name: "simple state machine",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				k, _ := kripkeModel(WithStateMachines(sm))
				_ = k.Solve()
				return k
			},
			want: `digraph {
  5438153399123815847 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:initial}

QueuedEvents:" ];
  12572739617557039328 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:initial}

QueuedEvents:
* testStateMachine<<EntryEvent;no fields" ];
  12572739617557039328 [ penwidth=5 ];
  12572739617557039328 -> 5438153399123815847;
}
`,
		},
		{
			name: "state machine with invariant violation",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = k.Solve()
				return k
			},
			want: `digraph {
  5438153399123815847 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:initial}

QueuedEvents:" ];
  5438153399123815847 [ color=red, penwidth=3 ];
  12572739617557039328 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:initial}

QueuedEvents:
* testStateMachine<<EntryEvent;no fields" ];
  12572739617557039328 [ penwidth=5 ];
  12572739617557039328 [ color=red, penwidth=3 ];
  12572739617557039328 -> 5438153399123815847;
}
`,
		},
		{
			name: "multiple state machines",
			setup: func() kripke {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				k, _ := kripkeModel(WithStateMachines(sm1, sm2))
				_ = k.Solve()
				return k
			},
			want: `digraph {
  2996475365393991777 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:state1}
* testStateMachine=no fields;{Name:Name,Type:string,Value:state2}

QueuedEvents:
* testStateMachine<<EntryEvent;no fields" ];
  7637389232411419170 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:state1}
* testStateMachine=no fields;{Name:Name,Type:string,Value:state2}

QueuedEvents:
* testStateMachine<<EntryEvent;no fields
* testStateMachine<<EntryEvent;no fields" ];
  7637389232411419170 [ penwidth=5 ];
  8000304505176841628 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:state1}
* testStateMachine=no fields;{Name:Name,Type:string,Value:state2}

QueuedEvents:" ];
  12282743545000744225 [ label="StateMachines:
* testStateMachine=no fields;{Name:Name,Type:string,Value:state1}
* testStateMachine=no fields;{Name:Name,Type:string,Value:state2}

QueuedEvents:
* testStateMachine<<EntryEvent;no fields" ];
  2996475365393991777 -> 8000304505176841628;
  7637389232411419170 -> 2996475365393991777;
  7637389232411419170 -> 12282743545000744225;
  12282743545000744225 -> 8000304505176841628;
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setup()
			var buf bytes.Buffer
			k.WriteAsDot(&buf)
			got := buf.String()

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("WriteAsDot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestKripke_WriteAsLog(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() kripke
		description string
		want        string
	}{
		{
			name: "no invariant violations",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = k.Solve()
				return k
			},
			description: "test invariant",
			want:        "No invariant violations found.\n",
		},
		{
			name: "with invariant violation",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = k.Solve()
				return k
			},
			description: "failing test invariant",
			want: `InvariantError:  failing test invariant   ✘
Path (length = 1):
  [0] <-- violation here
  StateMachines:
    Name: testStateMachine, Detail: no fields, State: {Name:Name,Type:string,Value:initial}
  QueuedEvents:
    StateMachine: testStateMachine, Event: EntryEvent, Detail: no fields
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setup()
			var buf bytes.Buffer
			k.WriteAsLog(&buf, tt.description)
			got := buf.String()

			if got != tt.want {
				t.Errorf("WriteAsLog() output mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestKripke_findPathsToViolations(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() kripke
		expectedPaths [][]worldID
	}{
		{
			name: "no violations",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				k, err := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = k.Solve()
				return k
			},
			expectedPaths: nil,
		},
		{
			name: "single violation",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				k, err := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = k.Solve()
				return k
			},
			expectedPaths: [][]worldID{{12572739617557039328}},
		},
		{
			name: "violation after transition",
			setup: func() kripke {
				// Create a simple state machine with transition that causes violation
				type testCounter struct {
					testStateMachine
					count int
				}

				spec := NewStateMachineSpec(&testCounter{})
				stateA := newTestState("A")
				stateB := newTestState("B")
				spec.DefineStates(stateA, stateB).SetInitialState(stateA)

				// On entry to state A, increment counter and go to B
				OnEntry(spec, stateA, func(ctx context.Context, sm *testCounter) {
					sm.count = 1
					Goto(ctx, stateB)
				})

				// On entry to state B, increment counter further
				OnEntry(spec, stateB, func(ctx context.Context, sm *testCounter) {
					sm.count = 2
				})

				sm := spec.NewInstance()

				// Invariant that fails when count > 1 (violated in state B)
				inv := NewInvariant(sm, func(sm *testCounter) bool {
					return sm.count <= 1
				})

				k, err := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				if err != nil {
					panic(err)
				}
				_ = k.Solve()
				return k
			},
			expectedPaths: [][]worldID{
				{7713272153197044482, 9676541489433535402, 14480809118815488961, 15722512986875548181, 8395799135532667686},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setup()
			actualPaths := k.findPathsToViolations()

			if diff := cmp.Diff(tt.expectedPaths, actualPaths); diff != "" {
				t.Errorf("Violation paths mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func TestWorld_label(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() world
		expected string
	}{
		{
			name: "single state machine with test state",
			setup: func() world {
				sm := newTestStateMachine(newTestState("test"))
				return initialWorld(sm)
			},
			expected: "StateMachines:\n* testStateMachine=no fields;{Name:Name,Type:string,Value:test}\n\nQueuedEvents:\n* testStateMachine<<EntryEvent;no fields",
		},
		{
			name: "single state machine with initial state",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				return initialWorld(sm)
			},
			expected: "StateMachines:\n* testStateMachine=no fields;{Name:Name,Type:string,Value:initial}\n\nQueuedEvents:\n* testStateMachine<<EntryEvent;no fields",
		},
		{
			name: "multiple state machines",
			setup: func() world {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				return initialWorld(sm1, sm2)
			},
			expected: "StateMachines:\n* testStateMachine=no fields;{Name:Name,Type:string,Value:state1}\n* testStateMachine=no fields;{Name:Name,Type:string,Value:state2}\n\nQueuedEvents:\n* testStateMachine<<EntryEvent;no fields\n* testStateMachine<<EntryEvent;no fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.setup()
			got := w.label()

			if got != tt.expected {
				t.Errorf("World.label() output mismatch\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestKripke_toWorldsData(t *testing.T) {
	tests := []struct {
		name           string
		setupKripke    func() kripke
		expectedWorlds []WorldJSON
	}{
		{
			name: "single state machine creates multiple worlds",
			setupKripke: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				k, _ := kripkeModel(WithStateMachines(sm))
				_ = k.Solve()
				return k
			},
			expectedWorlds: []WorldJSON{
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:initial}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{},
				},
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:initial}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "EntryEvent",
							Details:       "no fields",
						},
					},
				},
			},
		},
		{
			name: "multiple state machines creating multiple worlds",
			setupKripke: func() kripke {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				k, _ := kripkeModel(WithStateMachines(sm1, sm2))
				_ = k.Solve()
				return k
			},
			expectedWorlds: []WorldJSON{
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{},
				},
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "EntryEvent",
							Details:       "no fields",
						},
					},
				},
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "EntryEvent",
							Details:       "no fields",
						},
					},
				},
				{
					InvariantViolation: false,
					StateMachines: []StateMachineJSON{
						{
							ID:      "testStateMachine",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state1}",
							Details: "no fields",
						},
						{
							ID:      "testStateMachine_1",
							Name:    "testStateMachine",
							State:   "{Name:Name,Type:string,Value:state2}",
							Details: "no fields",
						},
					},
					QueuedEvents: []EventJSON{
						{
							TargetMachine: "testStateMachine",
							EventName:     "EntryEvent",
							Details:       "no fields",
						},
						{
							TargetMachine: "testStateMachine",
							EventName:     "EntryEvent",
							Details:       "no fields",
						},
					},
				},
			},
		},
		{
			name: "empty kripke structure",
			setupKripke: func() kripke {
				k, _ := kripkeModel()
				return k
			},
			expectedWorlds: []WorldJSON{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setupKripke()
			actualWorlds := k.toWorldsData()

			if diff := cmp.Diff(tt.expectedWorlds, actualWorlds); diff != "" {
				t.Errorf("Worlds data mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func TestKripke_Summarize(t *testing.T) {
	tests := []struct {
		name            string
		setupKripke     func() kripke
		executionTimeMs int64
		wantSummary     *kripkeSummary
	}{
		{
			name: "kripke with no invariant violations",
			setupKripke: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(true)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = k.Solve()
				return k
			},
			executionTimeMs: 150,
			wantSummary: &kripkeSummary{
				TotalWorlds:     2, // Actual world count after solving
				ExecutionTimeMs: 150,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0,
				},
			},
		},
		{
			name: "kripke with invariant violations",
			setupKripke: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				_ = k.Solve()
				return k
			},
			executionTimeMs: 250,
			wantSummary: &kripkeSummary{
				TotalWorlds:     2, // Actual world count after solving
				ExecutionTimeMs: 250,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: true,
					Count: 2, // All worlds have violations with BoolInvariant(false)
				},
			},
		},
		{
			name: "empty kripke structure",
			setupKripke: func() kripke {
				k, _ := kripkeModel()
				return k
			},
			executionTimeMs: 0,
			wantSummary: &kripkeSummary{
				TotalWorlds:     0,
				ExecutionTimeMs: 0,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0,
				},
			},
		},
		{
			name: "multiple worlds without violations",
			setupKripke: func() kripke {
				sm1 := newTestStateMachine(newTestState("state1"))
				sm2 := newTestStateMachine(newTestState("state2"))
				k, _ := kripkeModel(WithStateMachines(sm1, sm2))
				_ = k.Solve()
				return k
			},
			executionTimeMs: 500,
			wantSummary: &kripkeSummary{
				TotalWorlds:     4, // Actual world count
				ExecutionTimeMs: 500,
				InvariantViolations: struct {
					Found bool `json:"found"`
					Count int  `json:"count"`
				}{
					Found: false,
					Count: 0, // No violations without invariants
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setupKripke()
			summary := k.Summarize(tt.executionTimeMs)

			if !cmp.Equal(summary, tt.wantSummary) {
				t.Errorf("Summarize() mismatch: %v", cmp.Diff(tt.wantSummary, summary))
			}
		})
	}
}
