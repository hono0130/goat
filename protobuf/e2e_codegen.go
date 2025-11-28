package protobuf

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/goatx/goat/internal/e2egen"
	"github.com/goatx/goat/internal/strcase"
)

// codeGenerator generates Go test code from the intermediate representation.
// This is protocol-agnostic in terms of data structure, but generates
// protobuf/gRPC-specific Go code.
type codeGenerator struct {
	suite e2egen.TestSuite
}

// generateMainTest generates the main_test.go file with TestMain and global client variables.
func (g *codeGenerator) generateMainTest() (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("package %s\n\n", g.suite.PackageName))

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
	for _, svc := range g.suite.Services {
		if svc.ServicePackage != "" && !packagesSeen[svc.ServicePackage] {
			buf.WriteString(fmt.Sprintf("\tpb%s \"%s\"\n", strcase.ToSnakeCase(svc.ServiceName), svc.ServicePackage))
			packagesSeen[svc.ServicePackage] = true
		}
	}

	buf.WriteString(")\n\n")

	// Global client variables
	for _, svc := range g.suite.Services {
		if svc.ServicePackage != "" {
			buf.WriteString(fmt.Sprintf("var %s pb%s.%sClient\n", svc.ClientVarName, strcase.ToSnakeCase(svc.ServiceName), svc.ServiceName))
		}
	}
	buf.WriteString("\n")

	// TestMain function
	buf.WriteString("func TestMain(m *testing.M) {\n")

	// Start servers and initialize clients for each service
	for i, svc := range g.suite.Services {
		if svc.ServicePackage == "" {
			continue
		}

		serverVar := fmt.Sprintf("grpcServer%d", i)
		listenerVar := fmt.Sprintf("lis%d", i)
		connVar := fmt.Sprintf("conn%d", i)

		buf.WriteString(fmt.Sprintf("\t// Start %s server\n", svc.ServiceName))
		buf.WriteString(fmt.Sprintf("\t%s, err := net.Listen(\"tcp\", \"localhost:0\")\n", listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to listen: %%v\", err)\n")
		buf.WriteString("\t}\n\n")

		buf.WriteString(fmt.Sprintf("\t%s := grpc.NewServer()\n", serverVar))
		buf.WriteString("\t// TODO: Register your service implementation here\n")
		buf.WriteString(fmt.Sprintf("\t// pb%s.Register%sServer(%s, &yourServiceImplementation{})\n\n",
			strcase.ToSnakeCase(svc.ServiceName), svc.ServiceName, serverVar))

		buf.WriteString("\tgo func() {\n")
		buf.WriteString(fmt.Sprintf("\t\tif err := %s.Serve(%s); err != nil {\n", serverVar, listenerVar))
		buf.WriteString("\t\t\tlog.Fatalf(\"Failed to serve: %%v\", err)\n")
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}()\n\n")

		// Create client
		buf.WriteString(fmt.Sprintf("\t// Create %s client\n", svc.ServiceName))
		buf.WriteString(fmt.Sprintf("\t%s, err := grpc.Dial(%s.Addr().String(), grpc.WithInsecure())\n", connVar, listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to dial: %%v\", err)\n")
		buf.WriteString("\t}\n")
		buf.WriteString(fmt.Sprintf("\t%s = pb%s.New%sClient(%s)\n\n", svc.ClientVarName, strcase.ToSnakeCase(svc.ServiceName), svc.ServiceName, connVar))
	}

	// Run tests
	buf.WriteString("\t// Run tests\n")
	buf.WriteString("\tcode := m.Run()\n\n")

	// Cleanup
	buf.WriteString("\t// Cleanup\n")
	for i, svc := range g.suite.Services {
		if svc.ServicePackage == "" {
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
	buf.WriteString("func compareE2EOutput(expected, actual any) bool {\n")
	buf.WriteString("\treturn reflect.DeepEqual(expected, actual)\n")
	buf.WriteString("}\n")

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return buf.String(), fmt.Errorf("failed to format generated main test: %w", err)
	}

	return string(formatted), nil
}

// generateServiceTest generates a <service>_test.go file for a single service.
func (g *codeGenerator) generateServiceTest(svc e2egen.ServiceTestSuite) (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("package %s\n\n", g.suite.PackageName))

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"testing\"\n")

	if svc.ServicePackage != "" {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("\tpb%s \"%s\"\n", strcase.ToSnakeCase(svc.ServiceName), svc.ServicePackage))
	}

	buf.WriteString(")\n\n")

	// Generate test function for each method
	for _, method := range svc.Methods {
		testFunc, err := g.generateMethodTest(svc, method)
		if err != nil {
			return "", fmt.Errorf("failed to generate test for %s: %w", method.MethodName, err)
		}
		buf.WriteString(testFunc)
		buf.WriteString("\n\n")
	}

	// Format the generated code
	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		// If formatting fails, return unformatted code for debugging
		return buf.String(), fmt.Errorf("failed to format generated code: %w", err)
	}

	return string(formatted), nil
}

// generateMethodTest generates a table-driven test function for a single RPC method.
func (g *codeGenerator) generateMethodTest(svc e2egen.ServiceTestSuite, method e2egen.MethodTestSuite) (string, error) {
	if len(method.TestCases) == 0 {
		return "", fmt.Errorf("no test cases for method %s", method.MethodName)
	}

	firstCase := method.TestCases[0]
	pbAlias := "pb" + strcase.ToSnakeCase(svc.ServiceName)

	var buf strings.Builder

	// Comment
	buf.WriteString(fmt.Sprintf("// Test%s tests the %s RPC call.\n", method.MethodName, method.MethodName))
	buf.WriteString("// This test was automatically generated from model checking execution.\n")

	// Function definition
	buf.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {\n", method.MethodName))

	// Table definition
	buf.WriteString("\ttests := []struct {\n")
	buf.WriteString("\t\tname     string\n")
	buf.WriteString(fmt.Sprintf("\t\tinput    *%s.%s\n", pbAlias, firstCase.InputType))
	buf.WriteString(fmt.Sprintf("\t\texpected *%s.%s\n", pbAlias, firstCase.OutputType))
	buf.WriteString("\t}{\n")

	// Each test case
	for _, tc := range method.TestCases {
		buf.WriteString("\t\t{\n")
		buf.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", tc.Name))
		buf.WriteString(fmt.Sprintf("\t\t\tinput: %s,\n", e2egen.FormatStructLiteral(pbAlias, tc.InputType, tc.Input)))
		buf.WriteString(fmt.Sprintf("\t\t\texpected: %s,\n", e2egen.FormatStructLiteral(pbAlias, tc.OutputType, tc.Output)))
		buf.WriteString("\t\t},\n")
	}

	buf.WriteString("\t}\n\n")

	// Test loop
	buf.WriteString("\tfor _, tt := range tests {\n")
	buf.WriteString("\t\tt.Run(tt.name, func(t *testing.T) {\n")
	buf.WriteString("\t\t\tctx := context.Background()\n")
	buf.WriteString(fmt.Sprintf("\t\t\tactual, err := %s.%s(ctx, tt.input)\n",
		svc.ClientVarName, method.MethodName))
	buf.WriteString("\t\t\tif err != nil {\n")
	buf.WriteString("\t\t\t\tt.Fatalf(\"RPC call failed: %%v\", err)\n")
	buf.WriteString("\t\t\t}\n\n")
	buf.WriteString("\t\t\t// Verify the output matches expected\n")
	buf.WriteString("\t\t\tif !compareE2EOutput(tt.expected, actual) {\n")
	buf.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s output mismatch:\\nexpected: %%+v\\ngot:      %%+v\", tt.expected, actual)\n",
		method.MethodName))
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t})\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	return buf.String(), nil
}

