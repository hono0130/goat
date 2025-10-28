package goat

import "testing"

func TestEventuallyAlways(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cTrue := BoolCondition("c", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(EventuallyAlways(cTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Satisfied {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cFalse := BoolCondition("cF", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(EventuallyAlways(cFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Satisfied {
			t.Fatalf("expected rule violation")
		}
		if l, ok := res[0].Evidence.(*lasso); !ok || l == nil {
			t.Fatalf("expected lasso")
		}
	})
}

func TestAlwaysEventually(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cTrue := BoolCondition("c", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(AlwaysEventually(cTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Satisfied {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cFalse := BoolCondition("cF", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(AlwaysEventually(cFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Satisfied {
			t.Fatalf("expected rule violation")
		}
		if l, ok := res[0].Evidence.(*lasso); !ok || l == nil {
			t.Fatalf("expected lasso")
		}
	})
}

func TestWheneverPEventuallyQ(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		pTrue := BoolCondition("p", true)
		qTrue := BoolCondition("q", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(WheneverPEventuallyQ(pTrue, qTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Satisfied {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		pTrue := BoolCondition("p", true)
		qFalse := BoolCondition("q", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithRules(WheneverPEventuallyQ(pTrue, qFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Satisfied {
			t.Fatalf("expected rule violation")
		}
		if l, ok := res[0].Evidence.(*lasso); !ok || l == nil {
			t.Fatalf("expected lasso")
		}
	})
}
