package protobuf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoTestGenerator_Generate(t *testing.T) {
	testCase := &E2ETestCase{
		Name:        "user_service_test",
		Description: "Test user service operations",
		Traces: []RPCTrace{
			{
				MethodName: "CreateUser",
				InputType:  "CreateUserRequest",
				Input: map[string]any{
					"Username": "alice",
					"Email":    "alice@example.com",
				},
				OutputType: "CreateUserResponse",
				Output: map[string]any{
					"UserID":  "user_123",
					"Success": true,
				},
			},
		},
	}

	generator := NewGoTestGenerator("main")
	code, err := generator.Generate(testCase)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify generated code contains expected elements
	expectedElements := []string{
		"package main",
		"import",
		"\"testing\"",
		"\"reflect\"",
		"func TestUser_service_test_0_CreateUser(t *testing.T)",
		"CreateUserRequest",
		"CreateUserResponse",
		"\"alice\"",
		"\"alice@example.com\"",
		"\"user_123\"",
		"true",
		"compareE2EOutput",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected element: %q\nCode:\n%s", expected, code)
		}
	}
}

func TestGoTestGenerator_GenerateMultipleTraces(t *testing.T) {
	testCase := &E2ETestCase{
		Name: "multi_test",
		Traces: []RPCTrace{
			{
				MethodName: "Method1",
				InputType:  "Request1",
				Input:      map[string]any{"Field1": "value1"},
				OutputType: "Response1",
				Output:     map[string]any{"Result1": "output1"},
			},
			{
				MethodName: "Method2",
				InputType:  "Request2",
				Input:      map[string]any{"Field2": "value2"},
				OutputType: "Response2",
				Output:     map[string]any{"Result2": "output2"},
			},
		},
	}

	generator := NewGoTestGenerator("main")
	code, err := generator.Generate(testCase)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify both test functions are generated
	if !strings.Contains(code, "func TestMulti_test_0_Method1(t *testing.T)") {
		t.Error("Missing test function for Method1")
	}
	if !strings.Contains(code, "func TestMulti_test_1_Method2(t *testing.T)") {
		t.Error("Missing test function for Method2")
	}
}

func TestGoTestGenerator_GenerateToFile(t *testing.T) {
	testCase := &E2ETestCase{
		Name: "file_test",
		Traces: []RPCTrace{
			{
				MethodName: "TestMethod",
				InputType:  "TestRequest",
				Input:      map[string]any{"Data": "test"},
				OutputType: "TestResponse",
				Output:     map[string]any{"Result": "success"},
			},
		},
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "generated_test.go")

	generator := NewGoTestGenerator("main")
	err := generator.GenerateToFile(testCase, testFile)
	if err != nil {
		t.Fatalf("GenerateToFile() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Generated file does not exist")
	}

	// Read and verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), "package main") {
		t.Error("Generated file missing package declaration")
	}
}

func TestGoTestGenerator_AddImport(t *testing.T) {
	generator := NewGoTestGenerator("main")
	generator.AddImport("context")
	generator.AddImport("github.com/example/pkg")

	testCase := &E2ETestCase{
		Name: "import_test",
		Traces: []RPCTrace{
			{
				MethodName: "Test",
				InputType:  "Request",
				Input:      map[string]any{},
				OutputType: "Response",
				Output:     map[string]any{},
			},
		},
	}

	code, err := generator.Generate(testCase)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify custom imports are included
	if !strings.Contains(code, "\"context\"") {
		t.Error("Missing custom import: context")
	}
	if !strings.Contains(code, "\"github.com/example/pkg\"") {
		t.Error("Missing custom import: github.com/example/pkg")
	}
}

func TestFormatStructLiteral(t *testing.T) {
	generator := NewGoTestGenerator("main")

	tests := []struct {
		name     string
		typeName string
		data     map[string]any
		expected string
	}{
		{
			name:     "empty struct",
			typeName: "EmptyStruct",
			data:     map[string]any{},
			expected: "&EmptyStruct{}",
		},
		{
			name:     "single field",
			typeName: "SingleField",
			data: map[string]any{
				"Name": "test",
			},
			expected: "Name: \"test\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.formatStructLiteral(tt.typeName, tt.data)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("formatStructLiteral() result doesn't contain expected:\nexpected substring: %s\ngot: %s", tt.expected, result)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	generator := NewGoTestGenerator("main")

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "string",
			value:    "hello",
			expected: "\"hello\"",
		},
		{
			name:     "int",
			value:    42,
			expected: "42",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},
		{
			name:     "float",
			value:    3.14,
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.formatValue(tt.value)
			if result != tt.expected {
				t.Errorf("formatValue() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestToTestName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "simple_test",
			expected: "Simple_test",
		},
		{
			input:    "test-with-dashes",
			expected: "Test_with_dashes",
		},
		{
			input:    "test.with.dots",
			expected: "Test_with_dots",
		},
		{
			input:    "test with spaces",
			expected: "Test_with_spaces",
		},
		{
			input:    "CamelCaseTest",
			expected: "CamelCaseTest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTestName(tt.input)
			if result != tt.expected {
				t.Errorf("toTestName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGoTestGenerator_GenerateValidGoCode(t *testing.T) {
	testCase := &E2ETestCase{
		Name: "validation_test",
		Traces: []RPCTrace{
			{
				MethodName: "ValidateUser",
				InputType:  "ValidateRequest",
				Input: map[string]any{
					"UserID": "123",
					"Active": true,
				},
				OutputType: "ValidateResponse",
				Output: map[string]any{
					"Valid":  true,
					"Errors": nil,
				},
			},
		},
	}

	generator := NewGoTestGenerator("main")
	code, err := generator.Generate(testCase)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// The code should be formatted (go/format should succeed)
	// If Generate() returns without error, format.Source succeeded
	// So we just verify the code is not empty
	if len(code) == 0 {
		t.Error("Generated code is empty")
	}

	// Verify it's actually Go code by checking for package declaration
	if !strings.HasPrefix(code, "package main") {
		t.Error("Generated code doesn't start with package declaration")
	}
}
