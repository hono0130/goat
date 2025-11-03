package protobuf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestE2ETestRecorder_BasicUsage(t *testing.T) {
	recorder := NewE2ETestRecorder("test_recorder", "test description")

	// Register event types
	recorder.RegisterEventType(&TestRequest1{})
	recorder.RegisterEventType(&TestResponse1{})

	// Verify registration
	registry := recorder.GetEventRegistry()
	if len(registry) != 2 {
		t.Errorf("expected 2 registered types, got %d", len(registry))
	}

	// Record an RPC
	service := &TestService1{}
	input := &TestRequest1{Data: "test"}
	output := &TestResponse1{Result: "success"}

	err := recorder.Record("TestMethod", service, service, input, output, 1)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Get test case
	testCase := recorder.GetTestCase()
	if testCase.Name != "test_recorder" {
		t.Errorf("expected name 'test_recorder', got %s", testCase.Name)
	}
	if len(testCase.Traces) != 1 {
		t.Errorf("expected 1 trace, got %d", len(testCase.Traces))
	}
}

func TestE2ETestRecorder_GetContext(t *testing.T) {
	recorder := NewE2ETestRecorder("test", "test")
	ctx := recorder.GetContext(context.Background(), 123)

	// Verify trace recorder is in context
	if tr := getTraceRecorderFromContext(ctx); tr == nil {
		t.Error("expected trace recorder in context")
	}

	// Verify world ID is in context
	if worldID := getWorldIDFromContext(ctx); worldID != 123 {
		t.Errorf("expected world ID 123, got %d", worldID)
	}
}

func TestE2ETestRecorder_SaveToFile(t *testing.T) {
	recorder := NewE2ETestRecorder("file_test", "test file save")
	service := &TestService1{}

	_ = recorder.Record("TestMethod", service, service,
		&TestRequest1{Data: "input"},
		&TestResponse1{Result: "output"},
		1)

	// Save to temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_case.json")

	err := recorder.SaveToFile(testFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and is valid
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	// Verify we can load it back
	loaded, err := LoadTestCase(data)
	if err != nil {
		t.Fatalf("failed to load saved test case: %v", err)
	}

	if loaded.Name != "file_test" {
		t.Errorf("expected name 'file_test', got %s", loaded.Name)
	}
}

func TestE2ETestRunner_BasicUsage(t *testing.T) {
	runner := NewE2ETestRunner()

	// Register event types
	runner.RegisterEventType(&TestRequest1{})
	runner.RegisterEventType(&TestResponse1{})

	// Register state machine
	service := &TestService1{}
	runner.RegisterStateMachine("TestService1", service)

	// Set strict mode
	runner.SetStrictMode(true)

	// Create a test case
	testCase := &E2ETestCase{
		Name:        "test",
		Description: "test",
		Traces: []RPCTrace{
			{
				MethodName: "TestMethod",
				Sender:     "TestService1",
				Recipient:  "TestService1",
				InputType:  "TestRequest1",
				Input: map[string]any{
					"Data": "test input",
				},
				OutputType: "TestResponse1",
				Output: map[string]any{
					"Result": "test output",
				},
				WorldID: 1,
			},
		},
	}

	// Run the test (note: this will just create the events, not actually execute them)
	result := runner.Run(testCase)

	if result.TotalTraces != 1 {
		t.Errorf("expected 1 total trace, got %d", result.TotalTraces)
	}
}

func TestE2ETestRunner_RunFromFile(t *testing.T) {
	// Create a test case and save it
	testCase := &E2ETestCase{
		Name:        "file_test",
		Description: "test from file",
		Traces: []RPCTrace{
			{
				MethodName: "TestMethod",
				Sender:     "TestService1",
				Recipient:  "TestService1",
				InputType:  "TestRequest1",
				Input: map[string]any{
					"Data": "input",
				},
				OutputType: "TestResponse1",
				Output: map[string]any{
					"Result": "output",
				},
				WorldID: 1,
			},
		},
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")

	data, _ := SaveTestCase(testCase)
	if err := os.WriteFile(testFile, data, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create runner and run from file
	runner := NewE2ETestRunner()
	runner.RegisterEventType(&TestRequest1{})
	runner.RegisterEventType(&TestResponse1{})
	runner.RegisterStateMachine("TestService1", &TestService1{})

	result, err := runner.RunFromFile(testFile)
	if err != nil {
		t.Fatalf("RunFromFile failed: %v", err)
	}

	if result.TotalTraces != 1 {
		t.Errorf("expected 1 total trace, got %d", result.TotalTraces)
	}
}

func TestQuickRecord(t *testing.T) {
	service := &TestService1{}
	input := &TestRequest1{Data: "quick input"}
	output := &TestResponse1{Result: "quick output"}

	testCase := QuickRecord("quick_test", "QuickMethod", service, input, output)

	if testCase.Name != "quick_test" {
		t.Errorf("expected name 'quick_test', got %s", testCase.Name)
	}
	if len(testCase.Traces) != 1 {
		t.Errorf("expected 1 trace, got %d", len(testCase.Traces))
	}

	trace := testCase.Traces[0]
	if trace.MethodName != "QuickMethod" {
		t.Errorf("expected method 'QuickMethod', got %s", trace.MethodName)
	}
	if trace.Input["Data"] != "quick input" {
		t.Errorf("expected input 'quick input', got %v", trace.Input["Data"])
	}
	if trace.Output["Result"] != "quick output" {
		t.Errorf("expected output 'quick output', got %v", trace.Output["Result"])
	}
}

func TestContextIntegration(t *testing.T) {
	// Test WithTraceRecorder and retrieval
	recorder := NewTraceRecorder()
	ctx := context.Background()
	ctx = WithTraceRecorder(ctx, recorder)

	retrieved := getTraceRecorderFromContext(ctx)
	if retrieved != recorder {
		t.Error("failed to retrieve trace recorder from context")
	}

	// Test WithWorldID and retrieval
	ctx = WithWorldID(ctx, 456)
	worldID := getWorldIDFromContext(ctx)
	if worldID != 456 {
		t.Errorf("expected world ID 456, got %d", worldID)
	}

	// Test with no recorder
	emptyCtx := context.Background()
	if tr := getTraceRecorderFromContext(emptyCtx); tr != nil {
		t.Error("expected nil trace recorder from empty context")
	}

	// Test with no world ID
	if wid := getWorldIDFromContext(emptyCtx); wid != 0 {
		t.Errorf("expected 0 world ID from empty context, got %d", wid)
	}
}
