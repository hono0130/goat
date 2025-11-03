package protobuf

import (
	"reflect"
	"testing"

	"github.com/goatx/goat"
)

func TestCompareOutputs(t *testing.T) {
	tests := []struct {
		name         string
		expected     AbstractProtobufMessage
		actual       AbstractProtobufMessage
		strictMode   bool
		wantMatch    bool
		wantMismatch string
	}{
		{
			name: "exact match",
			expected: &TestResponse1{
				Result: "success",
			},
			actual: &TestResponse1{
				Result: "success",
			},
			strictMode:   true,
			wantMatch:    true,
			wantMismatch: "",
		},
		{
			name: "field mismatch",
			expected: &TestResponse1{
				Result: "success",
			},
			actual: &TestResponse1{
				Result: "failure",
			},
			strictMode:   true,
			wantMatch:    false,
			wantMismatch: "field Result mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, mismatch := CompareOutputs(tt.expected, tt.actual, tt.strictMode)

			if match != tt.wantMatch {
				t.Errorf("expected match=%v, got %v", tt.wantMatch, match)
			}

			if tt.wantMismatch != "" && mismatch == "" {
				t.Errorf("expected mismatch message, got empty string")
			}
			if tt.wantMismatch == "" && mismatch != "" {
				t.Errorf("expected no mismatch message, got: %s", mismatch)
			}
		})
	}
}

func TestCompareMaps(t *testing.T) {
	tests := []struct {
		name         string
		expected     map[string]any
		actual       map[string]any
		strictMode   bool
		wantMatch    bool
		wantMismatch string
	}{
		{
			name: "exact match",
			expected: map[string]any{
				"field1": "value1",
				"field2": int64(42),
			},
			actual: map[string]any{
				"field1": "value1",
				"field2": int64(42),
			},
			strictMode:   true,
			wantMatch:    true,
			wantMismatch: "",
		},
		{
			name: "missing field",
			expected: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			actual: map[string]any{
				"field1": "value1",
			},
			strictMode:   true,
			wantMatch:    false,
			wantMismatch: "missing field",
		},
		{
			name: "field count mismatch",
			expected: map[string]any{
				"field1": "value1",
			},
			actual: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			strictMode:   true,
			wantMatch:    false,
			wantMismatch: "field count mismatch",
		},
		{
			name: "value mismatch",
			expected: map[string]any{
				"field1": "expected",
			},
			actual: map[string]any{
				"field1": "actual",
			},
			strictMode:   true,
			wantMatch:    false,
			wantMismatch: "field field1 mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, mismatch := compareMaps(tt.expected, tt.actual, tt.strictMode)

			if match != tt.wantMatch {
				t.Errorf("expected match=%v, got %v", tt.wantMatch, match)
			}

			if tt.wantMismatch != "" && mismatch == "" {
				t.Errorf("expected mismatch containing '%s', got empty string", tt.wantMismatch)
			}
		})
	}
}

func TestValueEquals(t *testing.T) {
	tests := []struct {
		name  string
		a     any
		b     any
		equal bool
	}{
		{
			name:  "equal strings",
			a:     "hello",
			b:     "hello",
			equal: true,
		},
		{
			name:  "different strings",
			a:     "hello",
			b:     "world",
			equal: false,
		},
		{
			name:  "equal int64",
			a:     int64(42),
			b:     int64(42),
			equal: true,
		},
		{
			name:  "int and int64 same value",
			a:     int(42),
			b:     int64(42),
			equal: true,
		},
		{
			name:  "float and int same value",
			a:     float64(42),
			b:     int(42),
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueEquals(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("expected %v, got %v", tt.equal, result)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		kind     reflect.Kind
		expected bool
	}{
		{reflect.Int, true},
		{reflect.Int32, true},
		{reflect.Int64, true},
		{reflect.Float32, true},
		{reflect.Float64, true},
		{reflect.Uint, true},
		{reflect.String, false},
		{reflect.Bool, false},
		{reflect.Slice, false},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			result := isNumeric(tt.kind)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected float64
	}{
		{
			name:     "int",
			value:    int(42),
			expected: 42.0,
		},
		{
			name:     "int64",
			value:    int64(100),
			expected: 100.0,
		},
		{
			name:     "float32",
			value:    float32(3.14),
			expected: 3.14,
		},
		{
			name:     "float64",
			value:    float64(2.718),
			expected: 2.718,
		},
		{
			name:     "uint",
			value:    uint(50),
			expected: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.value)
			// Use approximate comparison for floats
			if diff := result - tt.expected; diff < -0.001 || diff > 0.001 {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateEventFromTrace(t *testing.T) {
	registry := map[string]reflect.Type{
		"TestRequest1": reflect.TypeOf(TestRequest1{}),
	}

	trace := RPCTrace{
		MethodName: "TestMethod",
		InputType:  "TestRequest1",
		Input: map[string]any{
			"Data": "test data",
		},
	}

	event, err := CreateEventFromTrace(trace, registry)
	if err != nil {
		t.Fatalf("CreateEventFromTrace failed: %v", err)
	}

	req, ok := event.(*TestRequest1)
	if !ok {
		t.Fatalf("expected *TestRequest1, got %T", event)
	}

	if req.Data != "test data" {
		t.Errorf("expected Data 'test data', got %s", req.Data)
	}
}

func TestCreateEventFromTrace_UnregisteredType(t *testing.T) {
	registry := map[string]reflect.Type{}

	trace := RPCTrace{
		InputType: "UnknownType",
	}

	_, err := CreateEventFromTrace(trace, registry)
	if err == nil {
		t.Error("expected error for unregistered type")
	}
}

func TestTestReplayResult_Summary(t *testing.T) {
	testCase := &E2ETestCase{
		Name:        "test_summary",
		Description: "test summary generation",
		Traces:      []RPCTrace{},
	}

	result := &TestReplayResult{
		TestCase:     testCase,
		Results:      []ReplayResult{},
		TotalTraces:  3,
		SuccessCount: 2,
		FailureCount: 1,
	}

	summary := result.Summary()

	// Check that summary contains expected information
	if !containsHelper(summary, "test_summary") {
		t.Error("summary should contain test name")
	}
	if !containsHelper(summary, "Total Traces: 3") {
		t.Error("summary should contain total traces")
	}
	if !containsHelper(summary, "Successful: 2") {
		t.Error("summary should contain success count")
	}
	if !containsHelper(summary, "Failed: 1") {
		t.Error("summary should contain failure count")
	}
}

func TestReplayTest(t *testing.T) {
	testCase := &E2ETestCase{
		Name: "replay_test",
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

	opts := ReplayOptions{
		EventRegistry: map[string]reflect.Type{
			"TestRequest1":  reflect.TypeOf(TestRequest1{}),
			"TestResponse1": reflect.TypeOf(TestResponse1{}),
		},
		StateMachines:      map[string]goat.AbstractStateMachine{},
		StrictMode:         true,
		StopOnFirstFailure: false,
	}

	result := ReplayTest(testCase, opts)

	if result.TotalTraces != 1 {
		t.Errorf("expected 1 total trace, got %d", result.TotalTraces)
	}

	if result.TestCase.Name != "replay_test" {
		t.Errorf("expected test case name 'replay_test', got %s", result.TestCase.Name)
	}
}
