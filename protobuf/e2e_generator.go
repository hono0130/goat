package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
)

// TestCase represents a single test case with input and output events.
type TestCase struct {
	// MethodName is the RPC method to test
	MethodName string

	// Input is the actual input event with field values populated
	Input AbstractProtobufMessage

	// GetOutput is a function that executes the handler and returns the output
	// This function should create a service instance, call the handler, and return the result
	GetOutput func() (AbstractProtobufMessage, error)
}

// E2ETestOptions configures E2E test generation.
type E2ETestOptions struct {
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
// For each test case, it calls GetOutput() to obtain the expected output by executing the handler,
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
//	    OutputDir: "./tests",
//	    PackageName: "main",
//	    Filename: "user_service_test.go",
//	    TestCases: []protobuf.TestCase{
//	        {
//	            MethodName: "CreateUser",
//	            Input: &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
//	            GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
//	                // Execute the handler to get the output
//	                svc := &UserService{}
//	                ctx := context.Background()
//	                resp := svc.CreateUser(ctx, &CreateUserRequest{Username: "alice", Email: "alice@example.com"})
//	                return resp, nil
//	            },
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

	// Generate test cases
	testCases := make([]E2ETestCase, 0, len(opts.TestCases))

	for i, tc := range opts.TestCases {
		// Execute GetOutput to get the expected output
		output, err := tc.GetOutput()
		if err != nil {
			return fmt.Errorf("test case %d (%s): failed to get output: %w", i, tc.MethodName, err)
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
