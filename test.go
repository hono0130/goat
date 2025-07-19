package goat

import (
	"io"
	"os"
)

// Test creates a Kripke model, solves it, and writes the log output.
// This is a convenience function that combines kripkeModel, Solve, and WriteAsLog.
func Test(opts ...Option) error {
	// Create Kripke model with provided options
	kripke, err := kripkeModel(opts...)
	if err != nil {
		return err
	}

	// Solve the model to explore all reachable states
	if err := kripke.Solve(); err != nil {
		return err
	}

	// Write results to stdout
	// TODO: In the future, we might want to collect descriptions from each invariant
	kripke.WriteAsLog(os.Stdout, "invariant violation")

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

	if err := kripke.Solve(); err != nil {
		return err
	}

	return kripke.WriteWorldsAsJSON(w)
}
