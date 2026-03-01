package main

import (
	"testing"

	"github.com/goatx/goat"
)

func TestSimpleNonDeterministic(t *testing.T) {
	opts := createSimpleNonDeterministicModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if result.Summary.TotalWorlds != 12 {
		t.Fatalf("expected TotalWorlds=12, got %d", result.Summary.TotalWorlds)
	}
}
