package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/goatx/goat"
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

// toSnakeCase converts a PascalCase or camelCase string to snake_case.
func toSnakeCase(s string) string {
	// Insert underscore before uppercase letters (except the first one)
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

// GenerateE2ETest generates Go test code from test cases.
// It generates:
//   - main_test.go: TestMain function with server setup and global client variables
//   - <service_name>_test.go: Test functions for each RPC method in the service
//
// For each test case, it automatically executes the registered handler to obtain the expected output,
// then generates Go test code.
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

	// Process each service
	var services []serviceInfo

	for si, svc := range opts.Services {
		// Get service name from spec
		serviceName := svc.Spec.GetServiceName()
		clientVarName := toSnakeCase(serviceName) + "Client"

		services = append(services, serviceInfo{
			Name:          serviceName,
			Package:       svc.ServicePackage,
			ClientVarName: clientVarName,
		})

		// Generate test cases by executing handlers for this service
		testCases := make([]E2ETestCase, 0)

		for mi, method := range svc.Methods {
			for ii, input := range method.Inputs {
				// Execute handler to get output
				output, err := executeHandler(svc.Spec, method.MethodName, input)
				if err != nil {
					return fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to execute handler: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				// Serialize input and output
				inputData, err := serializeMessage(input)
				if err != nil {
					return fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize input: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				outputData, err := serializeMessage(output)
				if err != nil {
					return fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize output: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				testCase := E2ETestCase{
					MethodName: method.MethodName,
					InputType:  getTypeName(input),
					Input:      inputData,
					OutputType: getTypeName(output),
					Output:     outputData,
				}

				testCases = append(testCases, testCase)
			}
		}

		// Generate service test file
		generator := NewGoTestGenerator(opts.PackageName)
		generator.ServiceName = serviceName
		generator.ServicePackage = svc.ServicePackage
		generator.ClientVarName = clientVarName

		code, err := generator.GenerateServiceTests(testCases)
		if err != nil {
			return fmt.Errorf("failed to generate test code for service %s: %w", serviceName, err)
		}

		// Write service test file
		filename := toSnakeCase(serviceName) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
			return fmt.Errorf("failed to write test file %s: %w", filename, err)
		}
	}

	// Generate main_test.go
	mainTestCode, err := generateMainTest(opts.PackageName, services)
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainTestPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainTestPath, []byte(mainTestCode), 0644); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
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
	ctx := goat.NewHandlerContext(instance)

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

// serviceInfo contains information about a service for main_test.go generation.
type serviceInfo struct {
	Name          string
	Package       string
	ClientVarName string
}

// generateMainTest generates the main_test.go file with TestMain and global client variables.
func generateMainTest(packageName string, services []serviceInfo) (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"log\"\n")
	buf.WriteString("\t\"net\"\n")
	buf.WriteString("\t\"os\"\n")
	buf.WriteString("\t\"reflect\"\n")
	buf.WriteString("\t\"testing\"\n")
	buf.WriteString("\n")
	buf.WriteString("\t\"google.golang.org/grpc\"\n")

	// Add imports for each service package
	packagesSeen := make(map[string]bool)
	for _, svc := range services {
		if svc.Package != "" && !packagesSeen[svc.Package] {
			buf.WriteString(fmt.Sprintf("\tpb%s \"%s\"\n", toSnakeCase(svc.Name), svc.Package))
			packagesSeen[svc.Package] = true
		}
	}

	buf.WriteString(")\n\n")

	// Global client variables
	for _, svc := range services {
		if svc.Package != "" {
			buf.WriteString(fmt.Sprintf("var %s pb%s.%sClient\n", svc.ClientVarName, toSnakeCase(svc.Name), svc.Name))
		}
	}
	buf.WriteString("\n")

	// TestMain function
	buf.WriteString("func TestMain(m *testing.M) {\n")

	// Start servers and initialize clients for each service
	for i, svc := range services {
		if svc.Package == "" {
			continue
		}

		serverVar := fmt.Sprintf("grpcServer%d", i)
		listenerVar := fmt.Sprintf("lis%d", i)
		connVar := fmt.Sprintf("conn%d", i)

		buf.WriteString(fmt.Sprintf("\t// Start %s server\n", svc.Name))
		buf.WriteString(fmt.Sprintf("\t%s, err := net.Listen(\"tcp\", \"localhost:0\")\n", listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to listen: %%v\", err)\n")
		buf.WriteString("\t}\n\n")

		buf.WriteString(fmt.Sprintf("\t%s := grpc.NewServer()\n", serverVar))
		buf.WriteString("\t// TODO: Register your service implementation here\n")
		buf.WriteString(fmt.Sprintf("\t// pb%s.Register%sServer(%s, &yourServiceImplementation{})\n\n",
			toSnakeCase(svc.Name), svc.Name, serverVar))

		buf.WriteString("\tgo func() {\n")
		buf.WriteString(fmt.Sprintf("\t\tif err := %s.Serve(%s); err != nil {\n", serverVar, listenerVar))
		buf.WriteString("\t\t\tlog.Fatalf(\"Failed to serve: %%v\", err)\n")
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}()\n\n")

		// Create client
		buf.WriteString(fmt.Sprintf("\t// Create %s client\n", svc.Name))
		buf.WriteString(fmt.Sprintf("\t%s, err := grpc.Dial(%s.Addr().String(), grpc.WithInsecure())\n", connVar, listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to dial: %%v\", err)\n")
		buf.WriteString("\t}\n")
		buf.WriteString(fmt.Sprintf("\t%s = pb%s.New%sClient(%s)\n\n", svc.ClientVarName, toSnakeCase(svc.Name), svc.Name, connVar))
	}

	// Run tests
	buf.WriteString("\t// Run tests\n")
	buf.WriteString("\tcode := m.Run()\n\n")

	// Cleanup
	buf.WriteString("\t// Cleanup\n")
	for i, svc := range services {
		if svc.Package == "" {
			continue
		}
		buf.WriteString(fmt.Sprintf("\tconn%d.Close()\n", i))
		buf.WriteString(fmt.Sprintf("\tgrpcServer%d.Stop()\n", i))
	}
	buf.WriteString("\n")

	buf.WriteString("\tos.Exit(code)\n")
	buf.WriteString("}\n\n")

	// Helper function
	buf.WriteString("// compareE2EOutput compares two values for equality in E2E tests.\n")
	buf.WriteString("// This is a helper function automatically generated for E2E testing.\n")
	buf.WriteString("func compareE2EOutput(expected, actual interface{}) bool {\n")
	buf.WriteString("\treturn reflect.DeepEqual(expected, actual)\n")
	buf.WriteString("}\n")

	return buf.String(), nil
}
