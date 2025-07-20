package goat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Test creates a Kripke model, solves it, and writes the log output.
// This is a convenience function that combines kripkeModel, Solve, and WriteAsLog.
func Test(opts ...Option) error {
	// Create Kripke model with provided options
	kripke, err := kripkeModel(opts...)
	if err != nil {
		return err
	}

	// Solve the model to explore all reachable states with timing
	start := time.Now()
	if err := kripke.Solve(); err != nil {
		return err
	}
	executionTime := time.Since(start).Milliseconds()

	kripke.WriteAsLog(os.Stdout, "invariant violation")

	// Print summary
	summary := kripke.Summarize(executionTime)
	_, _ = fmt.Fprintln(os.Stdout, "\nModel Checking Summary:")
	_, _ = fmt.Fprintf(os.Stdout, "Total Worlds: %d\n", summary.TotalWorlds)
	if summary.InvariantViolations.Found {
		_, _ = fmt.Fprintf(os.Stdout, "Invariant Violations: %d found\n", summary.InvariantViolations.Count)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Invariant Violations: None")
	}
	_, _ = fmt.Fprintf(os.Stdout, "Execution Time: %dms\n", summary.ExecutionTimeMs)

	return nil
}

func WithStateMachines(sms ...AbstractStateMachine) Option {
	return optionFunc(func(o *options) {
		o.sms = sms
	})
}

func WithInvariants(is ...Invariant) Option {
	return optionFunc(func(o *options) {
		o.invariants = is
	})
}

func Debug(w io.Writer, opts ...Option) error {
	kripke, err := kripkeModel(opts...)
	if err != nil {
		return err
	}

	start := time.Now()
	if err := kripke.Solve(); err != nil {
		return err
	}
	executionTime := time.Since(start).Milliseconds()

	worlds := kripke.toWorldsData()
	summary := kripke.Summarize(executionTime)

	result := map[string]any{
		"worlds":  worlds,
		"summary": summary,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
