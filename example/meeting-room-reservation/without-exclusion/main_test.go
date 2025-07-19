package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"testing"

// 	"github.com/goatx/goat"
// )

// func TestMeetingRoomReservationWithoutExclusion(t *testing.T) {
// 	_, _, _, _, _, opts := createMeetingRoomWithoutExclusionModel()

// 	var buf bytes.Buffer
// 	err := goat.Debug(&buf, opts...)
// 	if err != nil {
// 		t.Fatalf("Debug failed: %v", err)
// 	}

// 	fmt.Println(buf.String())

// 	var data map[string]interface{}
// 	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
// 		t.Fatalf("Failed to parse JSON: %v", err)
// 	}

// 	// Meeting room reservation without exclusion should have invariant violations
// 	worlds, ok := data["worlds"].([]interface{})
// 	if !ok {
// 		t.Fatalf("Expected worlds to be an array, got %T", data["worlds"])
// 	}

// 	if len(worlds) == 0 {
// 		t.Fatalf("Expected at least one world, got %d", len(worlds))
// 	}

// 	// This is the WITHOUT exclusion example, so we expect to find invariant violations
// 	foundViolation := false
// 	for i, world := range worlds {
// 		worldMap, ok := world.(map[string]interface{})
// 		if !ok {
// 			t.Fatalf("Expected world %d to be an object, got %T", i, world)
// 		}

// 		violation, ok := worldMap["invariant_violation"].(bool)
// 		if !ok {
// 			t.Fatalf("Expected invariant_violation to be a boolean in world %d, got %T", i, worldMap["invariant_violation"])
// 		}

// 		if violation {
// 			foundViolation = true
// 		}
// 	}

// 	if !foundViolation {
// 		t.Errorf("Expected to find invariant violations but none were found")
// 	}

// 	t.Logf("Meeting room reservation without exclusion test passed with %d worlds explored (invariant violations detected as expected)", len(worlds))
// }