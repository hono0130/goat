package goat

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestResult_String(t *testing.T) {
	tests := []struct {
		name       string
		result     *Result
		want       string
		wantSuffix string
		wantAbsent string
	}{
		{
			name: "no violations",
			result: &Result{
				Summary: Summary{TotalWorlds: 5, ExecutionTimeMs: 42},
			},
			want: `No violations found.

Model Checking Summary:
Total Worlds: 5
Execution Time: 42ms
`,
		},
		{
			name: "with invariant violation",
			result: &Result{
				Violations: []Violation{
					{Rule: "Always ok", Path: []WorldSnapshot{{}}},
				},
				Summary: Summary{TotalWorlds: 3, ExecutionTimeMs: 10},
			},
			wantSuffix: `
Model Checking Summary:
Total Worlds: 3
Execution Time: 10ms
`,
			wantAbsent: "No violations found.",
		},
		{
			name: "with temporal violation",
			result: &Result{
				Violations: []Violation{
					{Rule: "eventually always p", Path: []WorldSnapshot{{}}, Loop: []WorldSnapshot{{}}},
				},
				Summary: Summary{TotalWorlds: 7, ExecutionTimeMs: 99},
			},
			wantSuffix: `
Model Checking Summary:
Total Worlds: 7
Execution Time: 99ms
`,
			wantAbsent: "No violations found.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("String() mismatch\ngot:\n%s\nwant:\n%s", got, tt.want)
				}
				return
			}
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("String() missing summary suffix\ngot:\n%s\nwantSuffix:\n%s", got, tt.wantSuffix)
			}
			if strings.Contains(got, tt.wantAbsent) {
				t.Errorf("String() should not contain %q\ngot:\n%s", tt.wantAbsent, got)
			}
		})
	}
}

func TestTest(t *testing.T) {
	sm := StateMachineSnapshot{Name: "testStateMachine", State: "{Name:Name,Type:string,Value:s}", Details: "no fields"}
	entry := EventSnapshot{TargetMachine: "testStateMachine", EventName: "entryEvent", Details: "no fields"}

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
		want    *Result
	}{
		{
			name: "no violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("ok", true)))}
			}(),
			want: &Result{
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name: "invariant violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(Always(BoolCondition("bad", false)))}
			}(),
			want: &Result{
				Violations: []Violation{
					{
						Rule: "Always bad",
						Path: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{entry},
							},
						},
					},
				},
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name: "temporal violation",
			opts: func() []Option {
				sm := newTestStateMachine(newTestState("s"))
				return []Option{WithStateMachines(sm), WithRules(EventuallyAlways(BoolCondition("cF", false)))}
			}(),
			want: &Result{
				Violations: []Violation{
					{
						Rule: "eventually always cF",
						Path: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{entry},
							},
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{},
							},
						},
						Loop: []WorldSnapshot{
							{
								StateMachines: []StateMachineSnapshot{sm},
								QueuedEvents:  []EventSnapshot{},
							},
						},
					},
				},
				Summary: Summary{TotalWorlds: 2},
			},
		},
		{
			name:    "error on empty options",
			wantErr: true,
		},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(Summary{}, "ExecutionTimeMs"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Test(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Test() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, result, cmpOpts...); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
