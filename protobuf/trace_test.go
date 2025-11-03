package protobuf

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTraceRecorder_RecordRPC(t *testing.T) {
	recorder := NewTraceRecorder()
	sender := &TestService1{}
	recipient := &TestService1{}

	inputEvent := &TestRequest1{Data: "test input"}
	outputEvent := &TestResponse1{Result: "test output"}

	err := recorder.RecordRPC("TestMethod", sender, recipient, inputEvent, outputEvent, 123)
	if err != nil {
		t.Fatalf("RecordRPC failed: %v", err)
	}

	traces := recorder.GetTraces()
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	trace := traces[0]
	if trace.MethodName != "TestMethod" {
		t.Errorf("expected method name 'TestMethod', got %s", trace.MethodName)
	}
	if trace.InputType != "TestRequest1" {
		t.Errorf("expected input type 'TestRequest1', got %s", trace.InputType)
	}
	if trace.OutputType != "TestResponse1" {
		t.Errorf("expected output type 'TestResponse1', got %s", trace.OutputType)
	}
	if trace.WorldID != 123 {
		t.Errorf("expected world ID 123, got %d", trace.WorldID)
	}

	// Verify serialized data
	if trace.Input["Data"] != "test input" {
		t.Errorf("expected input data 'test input', got %v", trace.Input["Data"])
	}
	if trace.Output["Result"] != "test output" {
		t.Errorf("expected output result 'test output', got %v", trace.Output["Result"])
	}
}

func TestTraceRecorder_Clear(t *testing.T) {
	recorder := NewTraceRecorder()
	sender := &TestService1{}
	recipient := &TestService1{}

	_ = recorder.RecordRPC("TestMethod", sender, recipient, &TestRequest1{}, &TestResponse1{}, 1)
	_ = recorder.RecordRPC("TestMethod", sender, recipient, &TestRequest1{}, &TestResponse1{}, 2)

	if len(recorder.GetTraces()) != 2 {
		t.Fatalf("expected 2 traces before clear")
	}

	recorder.Clear()

	if len(recorder.GetTraces()) != 0 {
		t.Errorf("expected 0 traces after clear, got %d", len(recorder.GetTraces()))
	}
}

func TestTraceRecorder_ToTestCase(t *testing.T) {
	recorder := NewTraceRecorder()
	sender := &TestService1{}
	recipient := &TestService1{}

	_ = recorder.RecordRPC("Method1", sender, recipient, &TestRequest1{Data: "input1"}, &TestResponse1{Result: "output1"}, 1)
	_ = recorder.RecordRPC("Method2", sender, recipient, &TestRequest1{Data: "input2"}, &TestResponse1{Result: "output2"}, 2)

	testCase := recorder.ToTestCase("test_case", "test description")

	if testCase.Name != "test_case" {
		t.Errorf("expected name 'test_case', got %s", testCase.Name)
	}
	if testCase.Description != "test description" {
		t.Errorf("expected description 'test description', got %s", testCase.Description)
	}
	if len(testCase.Traces) != 2 {
		t.Errorf("expected 2 traces, got %d", len(testCase.Traces))
	}
}

func TestSerializeMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      AbstractProtobufMessage
		expected map[string]any
	}{
		{
			name: "simple message",
			msg: &TestRequest1{
				Data: "hello",
			},
			expected: map[string]any{
				"Data": "hello",
			},
		},
		{
			name: "message with multiple fields",
			msg: &TestResponse1{
				Result: "success",
			},
			expected: map[string]any{
				"Result": "success",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := serializeMessage(tt.msg)
			if err != nil {
				t.Fatalf("serializeMessage failed: %v", err)
			}

			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("serialization mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeserializeMessage(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		msg      AbstractProtobufMessage
		validate func(*testing.T, AbstractProtobufMessage)
	}{
		{
			name: "simple message",
			data: map[string]any{
				"Data": "hello",
			},
			msg: &TestRequest1{},
			validate: func(t *testing.T, msg AbstractProtobufMessage) {
				req := msg.(*TestRequest1)
				if req.Data != "hello" {
					t.Errorf("expected Data 'hello', got %s", req.Data)
				}
			},
		},
		{
			name: "message with multiple fields",
			data: map[string]any{
				"Result": "success",
			},
			msg: &TestResponse1{},
			validate: func(t *testing.T, msg AbstractProtobufMessage) {
				resp := msg.(*TestResponse1)
				if resp.Result != "success" {
					t.Errorf("expected Result 'success', got %s", resp.Result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deserializeMessage(tt.msg, tt.data)
			if err != nil {
				t.Fatalf("deserializeMessage failed: %v", err)
			}

			tt.validate(t, tt.msg)
		})
	}
}

func TestSaveAndLoadTestCase(t *testing.T) {
	// Create a test case
	original := &E2ETestCase{
		Name:        "test_case",
		Description: "test description",
		Traces: []RPCTrace{
			{
				MethodName: "TestMethod",
				Sender:     "TestService1@0x1234",
				Recipient:  "TestService1@0x5678",
				InputType:  "TestRequest1",
				Input: map[string]any{
					"Data": "test input",
				},
				OutputType: "TestResponse1",
				Output: map[string]any{
					"Result": "test output",
				},
				WorldID: 123,
			},
		},
	}

	// Save to JSON
	data, err := SaveTestCase(original)
	if err != nil {
		t.Fatalf("SaveTestCase failed: %v", err)
	}

	// Verify it's valid JSON
	var jsonCheck map[string]any
	if err := json.Unmarshal(data, &jsonCheck); err != nil {
		t.Fatalf("generated JSON is invalid: %v", err)
	}

	// Load back from JSON
	loaded, err := LoadTestCase(data)
	if err != nil {
		t.Fatalf("LoadTestCase failed: %v", err)
	}

	// Compare
	if diff := cmp.Diff(original, loaded); diff != "" {
		t.Errorf("save/load roundtrip mismatch (-want +got):\n%s", diff)
	}
}

func TestGetStateMachineID(t *testing.T) {
	sm1 := &TestService1{}
	sm2 := &TestService1{}

	id1 := getStateMachineID(sm1)
	id2 := getStateMachineID(sm2)

	// IDs should be different (different pointers)
	if id1 == id2 {
		t.Errorf("expected different IDs for different instances")
	}

	// ID should contain type name
	if !contains(id1, "TestService1") {
		t.Errorf("expected ID to contain type name, got %s", id1)
	}

	// Nil state machine
	nilID := getStateMachineID(nil)
	if nilID != "nil" {
		t.Errorf("expected 'nil' for nil state machine, got %s", nilID)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
