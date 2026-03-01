package main

import (
	"testing"

	"github.com/goatx/goat"
)

func TestSimpleHalt(t *testing.T) {
	opts := createSimpleHaltModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if result.Summary.TotalWorlds != 3 {
		t.Fatalf("expected TotalWorlds=3, got %d", result.Summary.TotalWorlds)
	}
}
