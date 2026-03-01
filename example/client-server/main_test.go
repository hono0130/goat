package main

import (
	"testing"

	"github.com/goatx/goat"
)

func TestClientServer(t *testing.T) {
	opts := createClientServerModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if result.Summary.TotalWorlds != 40 {
		t.Fatalf("expected TotalWorlds=40, got %d", result.Summary.TotalWorlds)
	}
}
