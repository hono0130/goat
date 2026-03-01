package goat

import (
	"strings"
	"testing"
)

func TestResultNoViolation(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cTrue := BoolCondition("ok", true)
	result, err := Test(
		WithStateMachines(sm),
		WithRules(Always(cTrue)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(result.Violations))
	}
	if result.Summary.TotalWorlds == 0 {
		t.Fatal("expected TotalWorlds > 0")
	}
	if !strings.Contains(result.String(), "No violations found.") {
		t.Fatalf("expected 'No violations found.' in output, got: %s", result.String())
	}
	if !strings.Contains(result.String(), "Model Checking Summary:") {
		t.Fatalf("expected summary in output, got: %s", result.String())
	}
}

func TestResultInvariantViolation(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cFalse := BoolCondition("bad", false)
	result, err := Test(
		WithStateMachines(sm),
		WithRules(Always(cFalse)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolation() {
		t.Fatal("expected violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}

	v := result.Violations[0]
	if v.Rule != "Always bad" {
		t.Fatalf("expected rule 'Always bad', got %q", v.Rule)
	}
	if len(v.Path) == 0 {
		t.Fatal("expected non-empty path")
	}
	if v.Loop != nil {
		t.Fatal("expected nil loop for invariant violation")
	}

	snap := v.Path[len(v.Path)-1]
	if len(snap.StateMachines) == 0 {
		t.Fatal("expected state machines in snapshot")
	}

	text := result.String()
	if !strings.Contains(text, "Condition failed. Not Always bad.") {
		t.Fatalf("expected violation text in output, got: %s", text)
	}
	if !strings.Contains(text, "Model Checking Summary:") {
		t.Fatalf("expected summary in output, got: %s", text)
	}
}

func TestResultTemporalViolation(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	cFalse := BoolCondition("cF", false)
	result, err := Test(
		WithStateMachines(sm),
		WithRules(EventuallyAlways(cFalse)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolation() {
		t.Fatal("expected violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}

	v := result.Violations[0]
	if v.Rule != "eventually always cF" {
		t.Fatalf("expected rule 'eventually always cF', got %q", v.Rule)
	}
	if len(v.Path) == 0 {
		t.Fatal("expected non-empty path (lasso prefix)")
	}
	if len(v.Loop) == 0 {
		t.Fatal("expected non-empty loop (lasso cycle)")
	}

	text := result.String()
	if !strings.Contains(text, "Condition failed. Not eventually always cF.") {
		t.Fatalf("expected violation text in output, got: %s", text)
	}
}

func TestResultSummary(t *testing.T) {
	sm := newTestStateMachine(newTestState("s"))
	result, err := Test(WithStateMachines(sm))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary.TotalWorlds == 0 {
		t.Fatal("expected TotalWorlds > 0")
	}
	if result.Summary.ExecutionTimeMs < 0 {
		t.Fatal("expected non-negative ExecutionTimeMs")
	}
}

func TestResultError(t *testing.T) {
	result, err := Test()
	if err == nil {
		t.Fatal("expected error for empty options")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}
