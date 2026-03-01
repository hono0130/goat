package main

import (
	"testing"

	"github.com/goatx/goat"
)

func TestTemporalRuleExample(t *testing.T) {
	opts := createTemporalRuleModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if result.Summary.TotalWorlds != 15 {
		t.Fatalf("expected TotalWorlds=15, got %d", result.Summary.TotalWorlds)
	}
}
