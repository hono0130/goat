package goat

import (
	"fmt"
	"sort"
	"strings"
)

// Result holds the outcome of a model checking run.
type Result struct {
	Violations []Violation
	Summary    Summary
	text       string
}

// HasViolation reports whether any violations were found.
func (r *Result) HasViolation() bool {
	return len(r.Violations) > 0
}

// String returns a human-readable report of the model checking results.
func (r *Result) String() string {
	return r.text
}

// Summary contains statistics about the model checking run.
type Summary struct {
	TotalWorlds     int
	ExecutionTimeMs int64
}

// Violation represents a single property violation found during model checking.
type Violation struct {
	Rule string
	Path []WorldSnapshot
	Loop []WorldSnapshot
}

// WorldSnapshot is a snapshot of the system state at one point in a trace.
type WorldSnapshot struct {
	StateMachines []StateMachineSnapshot
	QueuedEvents  []EventSnapshot
}

// StateMachineSnapshot is a snapshot of a single state machine.
type StateMachineSnapshot struct {
	Name    string
	State   string
	Details string
}

// EventSnapshot is a snapshot of a queued event.
type EventSnapshot struct {
	TargetMachine string
	EventName     string
	Details       string
}

func (m *model) buildResult(trResults []temporalRuleResult, executionTimeMs int64) *Result {
	result := &Result{
		Summary: Summary{
			TotalWorlds:     len(m.worlds),
			ExecutionTimeMs: executionTimeMs,
		},
	}

	if m.hasInvariantViolation {
		for _, w := range m.collectInvariantViolations() {
			name := w.condition.String()
			rule := "Always " + name
			if name == "" {
				rule = ""
			}
			result.Violations = append(result.Violations, Violation{
				Rule: rule,
				Path: m.buildWorldSnapshots(w.path),
			})
		}
	}

	for _, tr := range trResults {
		if tr.Satisfied {
			continue
		}
		l, ok := tr.Evidence.(*lasso)
		if !ok || l == nil {
			continue
		}
		result.Violations = append(result.Violations, Violation{
			Rule: tr.Rule,
			Path: m.buildWorldSnapshots(l.Prefix),
			Loop: m.buildWorldSnapshots(l.Loop),
		})
	}

	result.text = m.buildResultText(trResults, executionTimeMs)
	return result
}

func (m *model) buildWorldSnapshots(ids []worldID) []WorldSnapshot {
	snapshots := make([]WorldSnapshot, len(ids))
	for i, wid := range ids {
		w := m.worlds[wid]
		snapshots[i] = m.buildWorldSnapshot(w)
	}
	return snapshots
}

func (m *model) buildWorldSnapshot(w world) WorldSnapshot {
	smIDs := make([]string, 0, len(w.env.machines))
	for smID := range w.env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)

	sms := make([]StateMachineSnapshot, 0, len(smIDs))
	for _, smID := range smIDs {
		sm := w.env.machines[smID]
		sms = append(sms, StateMachineSnapshot{
			Name:    getStateMachineName(sm),
			State:   getStateDetails(sm.currentState()),
			Details: getStateMachineDetails(sm),
		})
	}

	events := make([]EventSnapshot, 0)
	for _, smID := range smIDs {
		if evts, ok := w.env.queue[smID]; ok {
			for _, evt := range evts {
				events = append(events, EventSnapshot{
					TargetMachine: getStateMachineName(w.env.machines[smID]),
					EventName:     getEventName(evt),
					Details:       getEventDetails(evt),
				})
			}
		}
	}

	return WorldSnapshot{
		StateMachines: sms,
		QueuedEvents:  events,
	}
}

func (m *model) buildResultText(trResults []temporalRuleResult, executionTimeMs int64) string {
	var sb strings.Builder

	if m.hasInvariantViolation {
		m.writeInvariantViolations(&sb)
	}
	if m.hasLTLViolation {
		m.writeTemporalViolations(&sb, trResults)
	}
	if !m.hasInvariantViolation && !m.hasLTLViolation {
		sb.WriteString("No violations found.\n")
	}

	summary := m.summarize(executionTimeMs)
	fmt.Fprintln(&sb, "\nModel Checking Summary:")
	fmt.Fprintf(&sb, "Total Worlds: %d\n", summary.TotalWorlds)
	fmt.Fprintf(&sb, "Execution Time: %dms\n", summary.ExecutionTimeMs)

	return sb.String()
}
