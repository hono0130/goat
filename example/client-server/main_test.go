package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestClientServer(t *testing.T) {
	_, _, opts := createClientServerModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	fmt.Println(buf.String())

	var data map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Load expected worlds from JSON file
	expectedWorldsData, err := os.ReadFile("expected_worlds.json")
	if err != nil {
		t.Fatalf("Failed to read expected worlds file: %v", err)
	}

	var expectedWorlds interface{}
	if err := json.Unmarshal(expectedWorldsData, &expectedWorlds); err != nil {
		t.Fatalf("Failed to parse expected worlds JSON: %v", err)
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreMapEntries(func(k, v interface{}) bool {
			key, ok := k.(string)
			return ok && key == "id"
		}),
	}

	if diff := cmp.Diff(expectedWorlds, data["worlds"], cmpOpts...); diff != "" {
		t.Errorf("Worlds mismatch (-expected +actual):\n%s", diff)
	}
}