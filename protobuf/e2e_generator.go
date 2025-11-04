package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/goatx/goat"
)

// TestCase represents a single test case with input event.
// The expected output is automatically calculated by executing the handler from the spec.
type TestCase struct {
	// MethodName is the RPC method to test
	MethodName string

	// Input is the actual input event with field values populated
	Input AbstractProtobufMessage
}

// E2ETestOptions configures E2E test generation.
type E2ETestOptions struct {
	// Spec is the protobuf service specification containing registered handlers
	Spec AbstractProtobufServiceSpec

	// OutputDir is the directory to save generated test files
	OutputDir string

	// PackageName is the package name for generated tests
	PackageName string

	// Filename is the name of the generated test file
	Filename string

	// TestCases are the test cases to generate
	TestCases []TestCase
}

// GenerateE2ETest generates Go test code from test cases.
// For each test case, it automatically executes the registered handler to obtain the expected output,
// then generates Go test code.
//
// Example:
//
//	spec := protobuf.NewProtobufServiceSpec(&UserService{})
//	protobuf.OnProtobufMessage(spec, idleState, "CreateUser",
//	    &CreateUserRequest{}, &CreateUserResponse{},
//	    func(ctx context.Context, req *CreateUserRequest, svc *UserService) protobuf.ProtobufResponse[*CreateUserResponse] {
//	        return protobuf.ProtobufSendTo(ctx, svc, &CreateUserResponse{UserID: "123", Success: true})
//	    })
//
//	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
//	    Spec: spec,
//	    OutputDir: "./tests",
//	    PackageName: "main",
//	    Filename: "user_service_test.go",
//	    TestCases: []protobuf.TestCase{
//	        {
//	            MethodName: "CreateUser",
//	            Input: &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
//	        },
//	    },
//	})
func GenerateE2ETest(opts E2ETestOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./tests"
	}
	if opts.PackageName == "" {
		opts.PackageName = "main"
	}
	if opts.Filename == "" {
		opts.Filename = "generated_e2e_test.go"
	}

	// Generate test cases by executing handlers
	testCases := make([]E2ETestCase, 0, len(opts.TestCases))

	for i, tc := range opts.TestCases {
		// Execute handler to get output
		output, err := executeHandler(opts.Spec, tc.MethodName, tc.Input)
		if err != nil {
			return fmt.Errorf("test case %d (%s): failed to execute handler: %w", i, tc.MethodName, err)
		}

		// Serialize input and output
		inputData, err := serializeMessage(tc.Input)
		if err != nil {
			return fmt.Errorf("test case %d (%s): failed to serialize input: %w", i, tc.MethodName, err)
		}

		outputData, err := serializeMessage(output)
		if err != nil {
			return fmt.Errorf("test case %d (%s): failed to serialize output: %w", i, tc.MethodName, err)
		}

		testCase := E2ETestCase{
			MethodName: tc.MethodName,
			InputType:  getTypeName(tc.Input),
			Input:      inputData,
			OutputType: getTypeName(output),
			Output:     outputData,
		}

		testCases = append(testCases, testCase)
	}

	// Generate Go test code
	generator := NewGoTestGenerator(opts.PackageName)

	// Generate code for all test cases
	code, err := generator.GenerateMultiple(testCases)
	if err != nil {
		return fmt.Errorf("failed to generate test code: %w", err)
	}

	// Write to file
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, opts.Filename)
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	return nil
}

// executeHandler executes a handler for the given method and input, returning the output event.
func executeHandler(spec AbstractProtobufServiceSpec, methodName string, input AbstractProtobufMessage) (AbstractProtobufMessage, error) {
	// Get handler for this method
	handlers := spec.GetHandlers()
	handler, ok := handlers[methodName]
	if !ok {
		return nil, fmt.Errorf("no handler found for method %s", methodName)
	}

	// Create state machine instance
	instance, err := spec.NewStateMachineInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine instance: %w", err)
	}

	// Create context for handler execution
	ctx := goat.NewTestContext(instance)

	// Call handler using reflection: handler(ctx, input, instance)
	handlerValue := reflect.ValueOf(handler)
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(input),
		reflect.ValueOf(instance),
	})

	// Extract event from ProtobufResponse using GetEvent()
	response := results[0]
	eventResults := response.MethodByName("GetEvent").Call(nil)

	return eventResults[0].Interface().(AbstractProtobufMessage), nil
}
