package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"testing"

// 	"github.com/goatx/goat"
// )

// func TestMeetingRoomReservationWithExclusion(t *testing.T) {
// 	_, _, _, _, opts := createMeetingRoomWithExclusionModel()

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

// 	// Meeting room reservation produces many worlds - verify basic structure and no invariant violations
// 	worlds, ok := data["worlds"].([]interface{})
// 	if !ok {
// 		t.Fatalf("Expected worlds to be an array, got %T", data["worlds"])
// 	}

// 	if len(worlds) == 0 {
// 		t.Fatalf("Expected at least one world, got %d", len(worlds))
// 	}

// 	// Since this is the WITH exclusion example, verify no invariant violations
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
// 			t.Errorf("Found unexpected invariant violation in world %d", i)
// 		}
// 	}

// 	t.Logf("Meeting room reservation with exclusion test passed with %d worlds explored (no invariant violations)", len(worlds))
// }