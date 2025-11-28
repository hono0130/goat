package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
)

// MethodTestCase represents test cases for a specific RPC method.
// Multiple inputs can be provided for the same method, and the expected output
// is automatically calculated by executing the handler from the spec.
type MethodTestCase struct {
	// MethodName is the RPC method to test
	MethodName string

	// Inputs are the actual input events with field values populated
	Inputs []AbstractProtobufMessage
}

// ServiceTestCase represents test cases for a specific service.
type ServiceTestCase struct {
	// Spec is the protobuf service specification containing registered handlers
	Spec AbstractProtobufServiceSpec

	// ServicePackage is the import path for the generated protobuf package (e.g., "github.com/example/proto/user")
	// If empty, no gRPC client code will be generated (TODO comments instead)
	ServicePackage string

	// Methods are the test cases for each RPC method
	Methods []MethodTestCase
}

// E2ETestOptions configures E2E test generation.
type E2ETestOptions struct {
	// OutputDir is the directory to save generated test files
	OutputDir string

	// PackageName is the package name for generated tests
	PackageName string

	// Services are the services to generate tests for
	Services []ServiceTestCase
}

// GenerateE2ETest generates Go test code from protobuf service specifications.
// It generates:
//   - main_test.go: TestMain function with server setup and global client variables
//   - <service_name>_test.go: Test functions for each RPC method in the service
//
// For each test case, it automatically executes the registered handler to obtain the expected output,
// then generates Go test code.
//
// This function orchestrates the entire generation process:
//  1. Build intermediate representation from protobuf specs (via buildTestSuite)
//  2. Generate Go test code from intermediate representation (via codeGenerator)
//  3. Write generated files to disk
//
// Example:
//
//	userSpec := protobuf.NewProtobufServiceSpec(&UserService{})
//	protobuf.OnProtobufMessage(userSpec, idleState, "CreateUser",
//	    &CreateUserRequest{}, &CreateUserResponse{},
//	    func(ctx context.Context, req *CreateUserRequest, svc *UserService) protobuf.ProtobufResponse[*CreateUserResponse] {
//	        return protobuf.ProtobufSendTo(ctx, svc, &CreateUserResponse{UserID: "123", Success: true})
//	    })
//
//	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
//	    OutputDir: "./tests",
//	    PackageName: "main",
//	    Services: []protobuf.ServiceTestCase{
//	        {
//	            Spec: userSpec,
//	            ServicePackage: "github.com/example/proto/user",
//	            Methods: []protobuf.MethodTestCase{
//	                {
//	                    MethodName: "CreateUser",
//	                    Inputs: []protobuf.AbstractProtobufMessage{
//	                        &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
//	                        &CreateUserRequest{Username: "bob", Email: "bob@example.com"},
//	                    },
//	                },
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

	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 1: Build intermediate representation from protobuf specs
	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	// Step 2: Generate Go test code from intermediate representation
	gen := &codeGenerator{suite: suite}

	// Generate main_test.go
	mainCode, err := gen.generateMainTest()
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainPath, []byte(mainCode), 0644); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	// Generate service test files
	for _, svc := range suite.Services {
		serviceCode, err := gen.generateServiceTest(svc)
		if err != nil {
			return fmt.Errorf("failed to generate test for %s: %w", svc.ServiceName, err)
		}

		filename := toSnakeCase(svc.ServiceName) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(serviceCode), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}
