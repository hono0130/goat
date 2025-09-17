package goat

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemporalRule_EventuallyAlways(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cTrue := BoolCondition("c", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(cTrue),
			WithTemporalRules(EventuallyAlways(cTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Holds {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cFalse := BoolCondition("cF", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(cFalse),
			WithTemporalRules(EventuallyAlways(cFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Holds {
			t.Fatalf("expected rule violation")
		}
		if res[0].Lasso == nil {
			t.Fatalf("expected lasso")
		}
	})
}

func TestTemporalRule_AlwaysEventually(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cTrue := BoolCondition("c", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(cTrue),
			WithTemporalRules(AlwaysEventually(cTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Holds {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		cFalse := BoolCondition("cF", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(cFalse),
			WithTemporalRules(AlwaysEventually(cFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Holds {
			t.Fatalf("expected rule violation")
		}
		if res[0].Lasso == nil {
			t.Fatalf("expected lasso")
		}
	})
}

func TestTemporalRule_WheneverPEventuallyQ(t *testing.T) {
	t.Run("holds", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		pTrue := BoolCondition("p", true)
		qTrue := BoolCondition("q", true)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(pTrue, qTrue),
			WithTemporalRules(WheneverPEventuallyQ(pTrue, qTrue)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if !res[0].Holds {
			t.Fatalf("expected rule to hold")
		}
	})

	t.Run("violation", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("s"))
		pTrue := BoolCondition("p", true)
		qFalse := BoolCondition("q", false)
		m, err := newModel(
			WithStateMachines(sm),
			WithConditions(pTrue, qFalse),
			WithTemporalRules(WheneverPEventuallyQ(pTrue, qFalse)),
		)
		if err != nil {
			t.Fatalf("newModel error: %v", err)
		}
		_ = m.Solve()
		res := m.checkLTL()
		if res[0].Holds {
			t.Fatalf("expected rule violation")
		}
		if res[0].Lasso == nil {
			t.Fatalf("expected lasso")
		}
	})
}

func TestTemporalRuleIntegration_Test(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cFalse := BoolCondition("c", false)
	err := Test(
		WithStateMachines(sm),
		WithConditions(cFalse),
		WithTemporalRules(EventuallyAlways(cFalse)),
	)
	if err == nil {
		t.Fatalf("expected error from Test")
	}
}

func TestTemporalRuleIntegration_Debug(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cFalse := BoolCondition("c", false)
	var buf bytes.Buffer
	err := Debug(&buf,
		WithStateMachines(sm),
		WithConditions(cFalse),
		WithTemporalRules(EventuallyAlways(cFalse)),
	)
	if err != nil {
		t.Fatalf("Debug error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "temporal_rules") {
		t.Fatalf("expected temporal_rules in debug output")
	}
}
