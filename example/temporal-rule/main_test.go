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

type temporalRule struct {
	Expression string `json:"expression"`
	Satisfied  bool   `json:"satisfied"`
	Evidence   any    `json:"evidence,omitempty"`
}

type debugOutput struct {
	Worlds        any            `json:"worlds"`
	TemporalRules []temporalRule `json:"temporal_rules"`
}

func TestTemporalRuleExample(t *testing.T) {
	opts := createTemporalRuleModel()

	var buf bytes.Buffer
	if err := goat.Debug(&buf, opts...); err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	var data debugOutput
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

	if diff := cmp.Diff(expectedWorlds, data.Worlds, cmpOpts...); diff != "" {
		t.Errorf("Worlds mismatch (-expected +actual):\n%s", diff)
	}

	if len(data.TemporalRules) != 1 {
		t.Fatalf("expected one temporal rule, got: %v", data.TemporalRules)
	}
	if !data.TemporalRules[0].Satisfied {
		t.Fatalf("expected temporal rule to hold")
	}

	expectedTemporalRulesData, err := os.ReadFile("expected_temporal_rules.json")
	if err != nil {
		t.Fatalf("Failed to read expected temporal rules JSON: %v", err)
	}

	var expectedTemporalRules []temporalRule
	if err := json.Unmarshal(expectedTemporalRulesData, &expectedTemporalRules); err != nil {
		t.Fatalf("Failed to parse expected temporal rules JSON: %v", err)
	}

	if diff := cmp.Diff(expectedTemporalRules, data.TemporalRules); diff != "" {
		t.Fatalf("temporal rules mismatch (-expected +actual):\n%s", diff)
	}
}
