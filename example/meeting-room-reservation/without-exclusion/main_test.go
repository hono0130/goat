package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMeetingRoomReservationWithoutExclusion(t *testing.T) {
	opts := createMeetingRoomWithoutExclusionModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	got, ok := data["summary"].(map[string]any)
	if !ok {
		t.Fatalf("Expected summary to be an object")
	}

	expectedSummary := map[string]any{
		"total_worlds": float64(12152),
		"invariant_violations": map[string]any{
			"found": true,
			"count": float64(2592),
		},
	}

	ignoreOpts := cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
		return k == "execution_time_ms"
	})

	if diff := cmp.Diff(expectedSummary, got, ignoreOpts); diff != "" {
		t.Errorf("Summary mismatch (-want +got):\n%s", diff)
	}
}

func TestSequenceDiagram(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "sequence.md")

	err := goat.AnalyzePackage(".", outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	want, err := os.ReadFile("sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
