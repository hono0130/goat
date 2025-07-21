package goat

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestKripke_Solve(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() kripke
		want    func() kripke
		wantErr bool
	}{
		{
			name: "simple state machine with no transitions",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				k, _ := kripkeModel(WithStateMachines(sm))
				return k
			},
			want: func() kripke {
				// Create the expected kripke structure
				sm := newTestStateMachine(newTestState("initial"))
				getInnerStateMachine(sm).smID = testStateMachineID

				// Initial world with EntryEvent in queue
				initialWorld := newWorld(Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&EntryEvent{}},
					},
				})

				// World after processing EntryEvent (queue becomes empty)
				processedWorld := newWorld(Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				})

				return kripke{
					worlds: worlds{
						initialWorld.id:   initialWorld,
						processedWorld.id: processedWorld,
					},
					initial: initialWorld,
					accessible: map[worldID][]worldID{
						initialWorld.id:   {processedWorld.id},
						processedWorld.id: {},
					},
					invariants: nil,
				}
			},
			wantErr: false,
		},
		{
			name: "state machine with invariant violation",
			setup: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				inv := BoolInvariant(false) // Always false invariant
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				return k
			},
			want: func() kripke {
				sm := newTestStateMachine(newTestState("initial"))
				getInnerStateMachine(sm).smID = testStateMachineID

				// Initial world with EntryEvent in queue (invariantViolation not set on initial)
				initialWorld := newWorld(Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&EntryEvent{}},
					},
				})
				// k.initial remains unchanged by Solve()
				initialWorld.invariantViolation = false

				// World after processing EntryEvent with invariant violation
				processedWorld := newWorld(Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {},
					},
				})
				processedWorld.invariantViolation = true

				// Copy initial world for worlds map (this one gets marked with violation)
				initialWorldInMap := initialWorld
				initialWorldInMap.invariantViolation = true

				return kripke{
					worlds: worlds{
						initialWorld.id:   initialWorldInMap, // This one has violation
						processedWorld.id: processedWorld,
					},
					initial: initialWorld, // This one stays without violation
					accessible: map[worldID][]worldID{
						initialWorld.id:   {processedWorld.id},
						processedWorld.id: {},
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.setup()
			err := k.Solve()
			if (err != nil) != tt.wantErr {
				t.Errorf("Solve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.want != nil {
				expected := tt.want()

				opts := cmp.Options{
					cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
					cmpopts.IgnoreFields(kripke{}, "invariants"), // Ignore function pointers
					cmp.AllowUnexported(kripke{}, world{}, Environment{}, StateMachine{}),
				}

				if diff := cmp.Diff(expected, k, opts); diff != "" {
					t.Errorf("Solve() result mismatch (-want +got):\n%s", diff)
				}
			}

			// Verify that initial world is explored
			if !k.worlds.member(k.initial) {
				t.Error("Initial world should be in explored worlds")
			}
		})
	}
}

func TestKripke_evaluateInvariants(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() (kripke, world)
		wantViolation bool
	}{
		{
			name: "no invariants",
			setup: func() (kripke, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				k, _ := kripkeModel(WithStateMachines(sm))
				w := initialWorld(sm)
				return k, w
			},
			wantViolation: false,
		},
		{
			name: "passing invariant",
			setup: func() (kripke, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv := BoolInvariant(true)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				w := initialWorld(sm)
				return k, w
			},
			wantViolation: false,
		},
		{
			name: "failing invariant",
			setup: func() (kripke, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv := BoolInvariant(false)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv),
				)
				w := initialWorld(sm)
				return k, w
			},
			wantViolation: true,
		},
		{
			name: "multiple invariants with one failing",
			setup: func() (kripke, world) {
				sm := newTestStateMachine(newTestState("initial"), newTestState("target"))
				inv1 := BoolInvariant(true)
				inv2 := BoolInvariant(false)
				inv3 := BoolInvariant(true)
				k, _ := kripkeModel(
					WithStateMachines(sm),
					WithInvariants(inv1, inv2, inv3),
				)
				w := initialWorld(sm)
				return k, w
			},
			wantViolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, w := tt.setup()
			result := k.evaluateInvariants(w)
			if result == tt.wantViolation {
				t.Errorf("evaluateInvariants() returned %v, but expected %v", result, !tt.wantViolation)
			}
		})
	}
}

func TestInitialWorld(t *testing.T) {
	tests := []struct {
		name string
		sms  []AbstractStateMachine
		want world
	}{
		{
			name: "single state machine",
			sms:  []AbstractStateMachine{newTestStateMachine(newTestState("initial"))},
			want: world{
				env: Environment{
					machines: map[string]AbstractStateMachine{
						"testStateMachine": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("initial"))
							getInnerStateMachine(sm).smID = testStateMachineID
							return sm
						}(),
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&EntryEvent{}},
					},
				},
				invariantViolation: false,
			},
		},
		{
			name: "multiple state machines",
			sms: []AbstractStateMachine{
				newTestStateMachine(newTestState("state1")),
				newTestStateMachine(newTestState("state2")),
			},
			want: world{
				env: Environment{
					machines: map[string]AbstractStateMachine{
						"testStateMachine": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("state1"))
							getInnerStateMachine(sm).smID = testStateMachineID
							return sm
						}(),
						"testStateMachine_1": func() AbstractStateMachine {
							sm := newTestStateMachine(newTestState("state2"))
							getInnerStateMachine(sm).smID = "testStateMachine_1"
							return sm
						}(),
					},
					queue: map[string][]AbstractEvent{
						"testStateMachine":   {&EntryEvent{}},
						"testStateMachine_1": {&EntryEvent{}},
					},
				},
				invariantViolation: false,
			},
		},
		{
			name: "no state machines",
			sms:  []AbstractStateMachine{},
			want: world{
				env: Environment{
					machines: map[string]AbstractStateMachine{},
					queue:    map[string][]AbstractEvent{},
				},
				invariantViolation: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initialWorld(tt.sms...)

			opts := cmp.Options{
				cmpopts.IgnoreFields(world{}, "id"),
				cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
				cmp.AllowUnexported(world{}, Environment{}, StateMachine{}),
			}

			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("initialWorld() mismatch (-want +got):\n%s", diff)
			}

			// Additional verification for EventHandlers and HandlerBuilders
			for smID, sm := range got.env.machines {
				innerSM := getInnerStateMachine(sm)

				// HandlerBuilders should be nil after initialization
				if innerSM.HandlerBuilders != nil {
					t.Errorf("StateMachine %q: HandlerBuilders should be nil after initialization, got %v", smID, innerSM.HandlerBuilders)
				}

				// EventHandlers should be initialized (not nil map)
				if innerSM.EventHandlers == nil {
					t.Errorf("StateMachine %q: EventHandlers should be initialized, got nil", smID)
				}
			}
		})
	}
}

// Test helper to create a handler that returns an error
type errorHandler struct{}

func (errorHandler) handle(_ Environment, _ string, _ AbstractEvent) ([]localState, error) {
	return nil, errors.New("test error")
}

func TestStepGlobal(t *testing.T) {

	tests := []struct {
		name    string
		setup   func() world
		want    func() []world
		wantErr bool
	}{
		{
			name: "returns error when handler fails",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID

				// Add a handler that returns an error
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)
				innerSM.EventHandlers[sm.currentState()] = []handlerInfo{
					{
						event:   &EntryEvent{},
						handler: errorHandler{},
					},
				}

				env := Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&EntryEvent{}},
					},
				}
				return newWorld(env)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns empty slice when no events to process",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				env := Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {}, // Empty queue
					},
				}
				return newWorld(env)
			},
			want: func() []world {
				return []world{} // Empty slice when no events to process
			},
			wantErr: false,
		},
		{
			name: "processes EntryEvent and returns new world",
			setup: func() world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				env := Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {&EntryEvent{}},
					},
				}
				return newWorld(env)
			},
			want: func() []world {
				sm := newTestStateMachine(newTestState("initial"))
				innerSM := getInnerStateMachine(sm)
				innerSM.smID = testStateMachineID
				innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)

				// Expected world after processing EntryEvent (queue becomes empty)
				expectedEnv := Environment{
					machines: map[string]AbstractStateMachine{
						testStateMachineID: sm,
					},
					queue: map[string][]AbstractEvent{
						testStateMachineID: {}, // Queue becomes empty after processing
					},
				}
				return []world{newWorld(expectedEnv)}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.setup()
			got, err := stepGlobal(w)

			if (err != nil) != tt.wantErr {
				t.Errorf("stepGlobal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && got != nil {
				t.Error("stepGlobal() should return nil slice on error")
				return
			}

			if tt.want != nil {
				expected := tt.want()

				opts := cmp.Options{
					cmpopts.IgnoreFields(world{}, "id"),
					cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
					cmp.AllowUnexported(world{}, Environment{}, StateMachine{}),
				}

				if diff := cmp.Diff(expected, got, opts); diff != "" {
					t.Errorf("stepGlobal() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
