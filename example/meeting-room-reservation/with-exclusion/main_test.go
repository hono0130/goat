package main

import (
	"testing"

	"github.com/goatx/goat"
)

func TestMeetingRoomReservationWithExclusion(t *testing.T) {
	opts := createMeetingRoomWithExclusionModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if result.HasViolation() {
		t.Fatal("expected no violations")
	}
	if result.Summary.TotalWorlds != 10606 {
		t.Fatalf("expected TotalWorlds=10606, got %d", result.Summary.TotalWorlds)
	}
}
