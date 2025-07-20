package goat

import (
	"bytes"
	"strings"
	"testing"
)

func TestKripke_WriteAsDot(t *testing.T) {
	tests := []struct {
		name string
		setup func() kripke
		checkContains []string
		checkNotContains []string
	}{
		{
			name: "simple state machine",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				k, _ := kripkeModel(WithStateMachines(sm))
				_ = k.Solve()
				return k
			},
			checkContains: []string{
				"digraph {",
				"}",
				"testStateMachine=no fields",
				"Value:initial",
				"[ penwidth=5 ]",
				"EntryEvent",
			},
			checkNotContains: []string{
				"color=red",
			},
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
			checkContains: []string{
				"digraph {",
				"}",
				"testStateMachine=no fields",
				"Value:initial",
				"[ penwidth=5 ]",
				"[ color=red, penwidth=3 ]",
				"EntryEvent",
			},
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
			checkContains: []string{
				"digraph {",
				"}",
				"testStateMachine=no fields",
				"Value:state1",
				"Value:state2",
				"[ penwidth=5 ]",
				"EntryEvent",
				"->", // Should have transitions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setup()
			var buf bytes.Buffer
			k.WriteAsDot(&buf)
			got := buf.String()
			
			for _, mustContain := range tt.checkContains {
				if !strings.Contains(got, mustContain) {
					t.Errorf("WriteAsDot() output missing required content %q\ngot:\n%s", mustContain, got)
				}
			}
			
			for _, mustNotContain := range tt.checkNotContains {
				if strings.Contains(got, mustNotContain) {
					t.Errorf("WriteAsDot() output contains forbidden content %q\ngot:\n%s", mustNotContain, got)
				}
			}
		})
	}
}

func TestKripke_WriteAsLog(t *testing.T) {
	tests := []struct {
		name string
		setup func() kripke
		description string
		want string
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
			want: "No invariant violations found.\n",
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

// func TestKripke_findPathsToViolations(t *testing.T) {
// 	tests := []struct {
// 		name        string
// 		setup       func() kripke
// 		expectPaths int
// 		pathLengths []int // Expected length of each path
// 	}{
// 		{
// 			name: "no violations",
// 			setup: func() kripke {
// 				sm := newTestStateMachine(newTestState("initial"))
// 				inv := BoolInvariant(true)
// 				k, err := kripkeModel(
// 					WithStateMachines(sm),
// 					WithInvariants(inv),
// 				)
// 				if err != nil {
// 					panic(err)
// 				}
// 				_ = k.Solve()
// 				return k
// 			},
// 			expectPaths: 0,
// 			pathLengths: []int{},
// 		},
// 		{
// 			name: "single violation",
// 			setup: func() kripke {
// 				sm := newTestStateMachine(newTestState("initial"))
// 				inv := BoolInvariant(false)
// 				k, err := kripkeModel(
// 					WithStateMachines(sm),
// 					WithInvariants(inv),
// 				)
// 				if err != nil {
// 					panic(err)
// 				}
// 				_ = k.Solve()
// 				return k
// 			},
// 			expectPaths: 1,
// 			pathLengths: []int{1}, // Path should have length 1 (initial world is violation)
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			k := tt.setup()
// 			paths := k.findPathsToViolations()
			
// 			// Check number of paths
// 			if len(paths) != tt.expectPaths {
// 				t.Errorf("Expected %d paths, got %d", tt.expectPaths, len(paths))
// 			}
			
// 			// Check path lengths
// 			if len(paths) != len(tt.pathLengths) {
// 				t.Errorf("Expected %d path lengths, got %d paths", len(tt.pathLengths), len(paths))
// 				return
// 			}
			
// 			for i, expectedLength := range tt.pathLengths {
// 				if len(paths[i]) != expectedLength {
// 					t.Errorf("Path %d: expected length %d, got %d", i, expectedLength, len(paths[i]))
// 				}
// 			}
			
// 			// Verify paths contain valid world IDs
// 			for i, path := range paths {
// 				for j, worldID := range path {
// 					if _, exists := k.worlds[worldID]; !exists {
// 						t.Errorf("Path %d[%d]: world ID %d does not exist in k.worlds", i, j, worldID)
// 					}
// 				}
				
// 				// Verify last world in path has invariant violation
// 				if len(path) > 0 {
// 					lastWorldID := path[len(path)-1]
// 					lastWorld := k.worlds[lastWorldID]
// 					if !lastWorld.invariantViolation {
// 						t.Errorf("Path %d: last world (ID %d) should have invariant violation", i, lastWorldID)
// 					}
// 				}
// 			}
// 		})
// 	}
// }

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