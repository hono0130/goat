package goat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

func Test(opts ...Option) error {
	kripke, err := kripkeModel(opts...)
	if err != nil {
		return err
	}

	start := time.Now()
	if err := kripke.Solve(); err != nil {
		return err
	}
	executionTime := time.Since(start).Milliseconds()

	kripke.writeLog(os.Stdout, "invariant violation")

	summary := kripke.summarize(executionTime)
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

	worlds := kripke.worldsToJSON()
	summary := kripke.summarize(executionTime)

	result := map[string]any{
		"worlds":  worlds,
		"summary": summary,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
