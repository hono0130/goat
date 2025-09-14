package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestClientServer(t *testing.T) {
	opts := createClientServerModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	expectedWorldsData, err := os.ReadFile("expected_worlds.json")
	if err != nil {
		t.Fatalf("Failed to read expected worlds file: %v", err)
	}

	var expectedWorlds any
	if err := json.Unmarshal(expectedWorldsData, &expectedWorlds); err != nil {
		t.Fatalf("Failed to parse expected worlds JSON: %v", err)
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreMapEntries(func(k, v any) bool {
			key, ok := k.(string)
			return ok && key == "summary"
		}),
	}

	if diff := cmp.Diff(expectedWorlds, data["worlds"], cmpOpts...); diff != "" {
		t.Errorf("Worlds mismatch (-expected +actual):\n%s", diff)
	}
}

func TestSequenceDiagram(t *testing.T) {
	var buf bytes.Buffer

	err := goat.AnalyzePackage(".", &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got := buf.String()

	want, err := os.ReadFile("sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if got != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
