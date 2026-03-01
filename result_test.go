package goat

import (
	"strings"
	"testing"
)

//nolint:gocyclo // gocyclo counts branches inside validate closures toward this function; actual loop body complexity is 3
func TestTest(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		wantErr  bool
		validate func(*testing.T, *Result)
	}{
		{
			name: "no violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("ok", true)))}
			}(),
			validate: func(t *testing.T, result *Result) {
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
			},
		},
		{
			name: "invariant violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("bad", false)))}
			}(),
			validate: func(t *testing.T, result *Result) {
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
			},
		},
		{
			name: "temporal violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(EventuallyAlways(BoolCondition("cF", false)))}
			}(),
			validate: func(t *testing.T, result *Result) {
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
			},
		},
		{
			name:    "error on empty options",
			wantErr: true,
			validate: func(t *testing.T, result *Result) {
				if result != nil {
					t.Fatal("expected nil result on error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Test(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Test() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
