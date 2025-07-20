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

// 	// Verify summary structure
// 	summary, ok := data["summary"].(map[string]interface{})
// 	if !ok {
// 		t.Fatalf("Expected summary to be an object, got %T", data["summary"])
// 	}

// 	// Check total worlds
// 	totalWorlds, ok := summary["total_worlds"].(float64)
// 	if !ok {
// 		t.Fatalf("Expected total_worlds to be a number, got %T", summary["total_worlds"])
// 	}

// 	if totalWorlds == 0 {
// 		t.Fatalf("Expected at least one world, got %f", totalWorlds)
// 	}
 
// 	// Check invariant violations - WITH exclusion should have NO violations
// 	violations, ok := summary["invariant_violations"].(map[string]interface{})
// 	if !ok {
// 		t.Fatalf("Expected invariant_violations to be an object, got %T", summary["invariant_violations"])
// 	}

// 	found, ok := violations["found"].(bool)
// 	if !ok {
// 		t.Fatalf("Expected found to be a boolean, got %T", violations["found"])
// 	}         

// 	count, ok := violations["count"].(float64)
// 	if !ok {
// 		t.Fatalf("Expected count to be a number, got %T", violations["count"])
// 	}

// 	// Since this is WITH exclusion, we expect NO violations
// 	if found {
// 		t.Errorf("Expected no invariant violations but found %f violations", count)
// 	}

// 	if count != 0 {
// 		t.Errorf("Expected violation count to be 0, got %f", count)
// 	}

// 	// Check execution time
// 	executionTime, ok := summary["execution_time_ms"].(float64)
// 	if !ok {
// 		t.Fatalf("Expected execution_time_ms to be a number, got %T", summary["execution_time_ms"])
// 	}

// 	if executionTime < 0 {
// 		t.Errorf("Expected execution time to be non-negative, got %f", executionTime)
// 	}

// 	t.Logf("Meeting room reservation WITH exclusion test passed:")
// 	t.Logf("  - Total worlds: %f", totalWorlds)
// 	t.Logf("  - Invariant violations: %t (count: %f)", found, count)
// 	t.Logf("  - Execution time: %fms", executionTime)
// }
