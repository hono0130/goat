package goat

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	stateInitial  = "initial"
	stateInitial1 = "initial1"
	stateInitial2 = "initial2"
)

func TestConditionFor(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*testStateMachine, world)
		checkFunc  func(*testStateMachine) bool
		wantResult bool
	}{
		{
			name: "always true invariant",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState(stateInitial))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				return true
			},
			wantResult: true,
		},
		{
			name: "always false invariant",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState(stateInitial))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				return false
			},
			wantResult: false,
		},
		{
			name: "check specific state",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState(stateInitial))
				return sm, newTestWorld(newTestEnvironment(sm))
			},
			checkFunc: func(sm *testStateMachine) bool {
				currentState := sm.currentState().(*testState)
				return currentState.Name == stateInitial
			},
			wantResult: true,
		},
		{
			name: "non-existent state machine",
			setup: func() (*testStateMachine, world) {
				sm := newTestStateMachine(newTestState(stateInitial))
				emptyEnv := environment{
					machines: make(map[string]AbstractStateMachine),
					queue:    make(map[string][]AbstractEvent),
				}
				return sm, newTestWorld(emptyEnv)
			},
			checkFunc: func(sm *testStateMachine) bool {
				return true
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, w := tt.setup()
			cond := NewCondition("test", sm, tt.checkFunc)
			got := cond.Evaluate(w)
			if diff := cmp.Diff(tt.wantResult, got); diff != "" {
				t.Errorf("Evaluate() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBoolCondition(t *testing.T) {
	tests := []struct {
		name       string
		value      bool
		wantResult bool
	}{
		{
			name:       "true invariant",
			value:      true,
			wantResult: true,
		},
		{
			name:       "false invariant",
			value:      false,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newTestWorld(newTestEnvironment())
			cond := BoolCondition("bool", tt.value)

			got := cond.Evaluate(w)
			if diff := cmp.Diff(tt.wantResult, got); diff != "" {
				t.Errorf("Evaluate() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewMultiCondition(t *testing.T) {
	t.Run("basic two machines", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState(stateInitial))
		sm2 := newTestStateMachine(newTestState(stateInitial))
		w := initialWorld(sm1, sm2)

		inv := NewMultiCondition("multi", func(ms Machines) bool {
			m1, ok := GetMachine(ms, sm1)
			if !ok {
				return false
			}
			m2, ok := GetMachine(ms, sm2)
			if !ok {
				return false
			}
			s1 := m1.currentState().(*testState)
			s2 := m2.currentState().(*testState)
			return s1.Name == stateInitial && s2.Name == stateInitial
		}, sm1, sm2)

		if got := inv.Evaluate(w); !got {
			t.Errorf("Evaluate() = false, want true")
		}
	})

	t.Run("missing machine returns false", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState(stateInitial))
		type testStateMachineB struct{ StateMachine }
		specB := NewStateMachineSpec(&testStateMachineB{})
		s := newTestState(stateInitial)
		specB.DefineStates(s).SetInitialState(s)
		smB, _ := specB.NewInstance()

		w := initialWorld(sm1)

		inv := NewMultiCondition("missing", func(ms Machines) bool {
			_, ok1 := GetMachine(ms, sm1)
			_, ok2 := GetMachine(ms, smB)
			return ok1 && ok2
		}, sm1, smB)

		if got := inv.Evaluate(w); got {
			t.Errorf("Evaluate() = true, want false")
		}
	})

	t.Run("typed access and missing detection", func(t *testing.T) {
		sm := newTestStateMachine(newTestState(stateInitial))
		type testStateMachineC struct{ StateMachine }
		specC := NewStateMachineSpec(&testStateMachineC{})
		s := newTestState(stateInitial)
		specC.DefineStates(s).SetInitialState(s)
		other, _ := specC.NewInstance()

		w := initialWorld(sm)

		inv := NewMultiCondition("typed", func(ms Machines) bool {
			got, ok := GetMachine(ms, sm)
			if !ok {
				return false
			}
			if _, ok := got.currentState().(*testState); !ok {
				return false
			}
			if _, ok := GetMachine(ms, other); ok {
				return false
			}
			return true
		}, sm)

		if got := inv.Evaluate(w); !got {
			t.Errorf("Evaluate() = false, want true")
		}
	})

	t.Run("two instances from same spec", func(t *testing.T) {
		spec := NewStateMachineSpec(&testStateMachine{})
		s := newTestState(stateInitial)
		spec.DefineStates(s).SetInitialState(s)
		m1, _ := spec.NewInstance()
		m2, _ := spec.NewInstance()

		w := initialWorld(m1, m2)

		inv := NewMultiCondition("same-spec", func(ms Machines) bool {
			x1, ok1 := GetMachine(ms, m1)
			if !ok1 {
				return false
			}
			x2, ok2 := GetMachine(ms, m2)
			if !ok2 {
				return false
			}
			if x1.id() == x2.id() {
				return false
			}
			s1 := x1.currentState().(*testState)
			s2 := x2.currentState().(*testState)
			return s1.Name == stateInitial && s2.Name == stateInitial
		}, m1, m2)

		if got := inv.Evaluate(w); !got {
			t.Errorf("Evaluate() = false, want true")
		}
	})
}

func TestNewCondition2(t *testing.T) {
	t.Run("basic two machines", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState(stateInitial1))
		sm2 := newTestStateMachine(newTestState(stateInitial2))
		w := initialWorld(sm1, sm2)

		inv := NewCondition2("pair", sm1, sm2, func(m1 *testStateMachine, m2 *testStateMachine) bool {
			s1 := m1.currentState().(*testState)
			s2 := m2.currentState().(*testState)
			return s1.Name == stateInitial1 && s2.Name == stateInitial2
		})

		if got := inv.Evaluate(w); !got {
			t.Errorf("Evaluate() = false, want true")
		}
	})

	t.Run("missing second machine returns false", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState("initial"))

		type testStateMachineB struct{ StateMachine }
		specB := NewStateMachineSpec(&testStateMachineB{})
		s := newTestState("x")
		specB.DefineStates(s).SetInitialState(s)
		smB, _ := specB.NewInstance()

		w := initialWorld(sm1)

		inv := NewCondition2("missing", sm1, smB, func(_ *testStateMachine, _ *testStateMachineB) bool { return true })

		if got := inv.Evaluate(w); got {
			t.Errorf("Evaluate() = true, want false")
		}
	})
}

func TestNewCondition3(t *testing.T) {
	t.Run("basic three machines", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState("a"))
		sm2 := newTestStateMachine(newTestState("b"))
		sm3 := newTestStateMachine(newTestState("c"))
		w := initialWorld(sm1, sm2, sm3)

		inv := NewCondition3("triple", sm1, sm2, sm3, func(x *testStateMachine, y *testStateMachine, z *testStateMachine) bool {
			sx := x.currentState().(*testState)
			sy := y.currentState().(*testState)
			sz := z.currentState().(*testState)
			return sx.Name == "a" && sy.Name == "b" && sz.Name == "c"
		})

		if got := inv.Evaluate(w); !got {
			t.Errorf("Evaluate() = false, want true")
		}
	})

	t.Run("missing one of three returns false", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState("a"))
		sm2 := newTestStateMachine(newTestState("b"))

		type testStateMachineC struct{ StateMachine }
		specC := NewStateMachineSpec(&testStateMachineC{})
		s := newTestState("c")
		specC.DefineStates(s).SetInitialState(s)
		smC, _ := specC.NewInstance()

		w := initialWorld(sm1, sm2)

		inv := NewCondition3("missing", sm1, sm2, smC, func(_ *testStateMachine, _ *testStateMachine, _ *testStateMachineC) bool { return true })

		if got := inv.Evaluate(w); got {
			t.Errorf("Evaluate() = true, want false")
		}
	})
}

func TestConvenienceConditions_Integration(t *testing.T) {
	t.Run("combination of NewCondition and NewCondition2", func(t *testing.T) {
		sm1 := newTestStateMachine(newTestState(stateInitial1))
		sm2 := newTestStateMachine(newTestState(stateInitial2))

		condSingle := NewCondition("single", sm1, func(m *testStateMachine) bool {
			return m.currentState().(*testState).Name == "initial1"
		})

		condPair := NewCondition2("pair", sm1, sm2, func(m1 *testStateMachine, m2 *testStateMachine) bool {
			return m1.currentState().(*testState).Name == stateInitial1 && m2.currentState().(*testState).Name == stateInitial2
		})

		m, err := newModel(
			WithStateMachines(sm1, sm2),
			WithRules(Always(condSingle), Always(condPair)),
		)
		if err != nil {
			t.Fatalf("newModel() error: %v", err)
		}

		if ok := m.evaluateInvariants(m.initial); !ok {
			t.Errorf("evaluateInvariants() = false, want true")
		}
	})
}
