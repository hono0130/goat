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

	// Get handlers from spec
	handlers := opts.Spec.GetHandlers()
	if handlers == nil {
		return fmt.Errorf("no handlers found in spec")
	}

	// Get the state machine spec to create instances
	spec := opts.Spec.GetSpec()
	if spec == nil {
		return fmt.Errorf("no state machine spec found")
	}

	// Generate test cases
	testCases := make([]E2ETestCase, 0, len(opts.TestCases))

	for i, tc := range opts.TestCases {
		// Look up handler for this method
		handler, ok := handlers[tc.MethodName]
		if !ok {
			return fmt.Errorf("test case %d: no handler found for method %s", i, tc.MethodName)
		}

		// Execute handler to get output
		output, err := executeHandler(handler, tc.Input, spec)
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

// executeHandler executes a handler function using reflection and returns the output event.
// The handler has signature: func(context.Context, I, T) ProtobufResponse[O]
func executeHandler(handler any, input AbstractProtobufMessage, spec any) (AbstractProtobufMessage, error) {
	// Get the handler function value
	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler is not a function")
	}

	// Create a state machine instance using reflection
	// Call spec.NewInstance() to get a proper instance
	specValue := reflect.ValueOf(spec)
	newInstanceMethod := specValue.MethodByName("NewInstance")
	if !newInstanceMethod.IsValid() {
		return nil, fmt.Errorf("spec does not have NewInstance method")
	}

	results := newInstanceMethod.Call(nil)
	if len(results) != 2 {
		return nil, fmt.Errorf("NewInstance returned %d values, expected 2 (instance, error)", len(results))
	}

	// Check for error
	if !results[1].IsNil() {
		return nil, fmt.Errorf("failed to create state machine instance: %v", results[1].Interface())
	}

	instance := results[0].Interface().(goat.AbstractStateMachine)

	// Create a test context with environment for handler execution
	ctx := goat.NewTestContext(instance)

	// Call the handler with (ctx, input, instance)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(input),
		reflect.ValueOf(instance),
	}

	handlerResults := handlerValue.Call(args)
	if len(handlerResults) != 1 {
		return nil, fmt.Errorf("handler returned %d values, expected 1", len(handlerResults))
	}

	// The result is a ProtobufResponse[O]
	// Call GetEvent() to extract the event
	response := handlerResults[0]
	if !response.IsValid() {
		return nil, fmt.Errorf("handler returned invalid value")
	}

	// Call GetEvent() method to get the event
	getEventMethod := response.MethodByName("GetEvent")
	if !getEventMethod.IsValid() {
		return nil, fmt.Errorf("response does not have GetEvent method")
	}

	eventResults := getEventMethod.Call(nil)
	if len(eventResults) != 1 {
		return nil, fmt.Errorf("GetEvent returned %d values, expected 1", len(eventResults))
	}

	// Convert to AbstractProtobufMessage
	output, ok := eventResults[0].Interface().(AbstractProtobufMessage)
	if !ok {
		return nil, fmt.Errorf("event is not an AbstractProtobufMessage")
	}

	return output, nil
}
