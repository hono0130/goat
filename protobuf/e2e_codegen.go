package protobuf

import (
	"fmt"
	"go/format"
	"reflect"
	"sort"
	"strings"
)

// codeGenerator generates Go test code from the intermediate representation.
// This is protocol-agnostic in terms of data structure, but generates
// protobuf/gRPC-specific Go code.
type codeGenerator struct {
	suite testSuite
}

// generateMainTest generates the main_test.go file with TestMain and global client variables.
func (g *codeGenerator) generateMainTest() (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("package %s\n\n", g.suite.packageName))

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
	for _, svc := range g.suite.services {
		if svc.servicePackage != "" && !packagesSeen[svc.servicePackage] {
			buf.WriteString(fmt.Sprintf("\tpb%s \"%s\"\n", toSnakeCase(svc.serviceName), svc.servicePackage))
			packagesSeen[svc.servicePackage] = true
		}
	}

	buf.WriteString(")\n\n")

	// Global client variables
	for _, svc := range g.suite.services {
		if svc.servicePackage != "" {
			buf.WriteString(fmt.Sprintf("var %s pb%s.%sClient\n", svc.clientVarName, toSnakeCase(svc.serviceName), svc.serviceName))
		}
	}
	buf.WriteString("\n")

	// TestMain function
	buf.WriteString("func TestMain(m *testing.M) {\n")

	// Start servers and initialize clients for each service
	for i, svc := range g.suite.services {
		if svc.servicePackage == "" {
			continue
		}

		serverVar := fmt.Sprintf("grpcServer%d", i)
		listenerVar := fmt.Sprintf("lis%d", i)
		connVar := fmt.Sprintf("conn%d", i)

		buf.WriteString(fmt.Sprintf("\t// Start %s server\n", svc.serviceName))
		buf.WriteString(fmt.Sprintf("\t%s, err := net.Listen(\"tcp\", \"localhost:0\")\n", listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to listen: %%v\", err)\n")
		buf.WriteString("\t}\n\n")

		buf.WriteString(fmt.Sprintf("\t%s := grpc.NewServer()\n", serverVar))
		buf.WriteString("\t// TODO: Register your service implementation here\n")
		buf.WriteString(fmt.Sprintf("\t// pb%s.Register%sServer(%s, &yourServiceImplementation{})\n\n",
			toSnakeCase(svc.serviceName), svc.serviceName, serverVar))

		buf.WriteString("\tgo func() {\n")
		buf.WriteString(fmt.Sprintf("\t\tif err := %s.Serve(%s); err != nil {\n", serverVar, listenerVar))
		buf.WriteString("\t\t\tlog.Fatalf(\"Failed to serve: %%v\", err)\n")
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}()\n\n")

		// Create client
		buf.WriteString(fmt.Sprintf("\t// Create %s client\n", svc.serviceName))
		buf.WriteString(fmt.Sprintf("\t%s, err := grpc.Dial(%s.Addr().String(), grpc.WithInsecure())\n", connVar, listenerVar))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tlog.Fatalf(\"Failed to dial: %%v\", err)\n")
		buf.WriteString("\t}\n")
		buf.WriteString(fmt.Sprintf("\t%s = pb%s.New%sClient(%s)\n\n", svc.clientVarName, toSnakeCase(svc.serviceName), svc.serviceName, connVar))
	}

	// Run tests
	buf.WriteString("\t// Run tests\n")
	buf.WriteString("\tcode := m.Run()\n\n")

	// Cleanup
	buf.WriteString("\t// Cleanup\n")
	for i, svc := range g.suite.services {
		if svc.servicePackage == "" {
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
func (g *codeGenerator) generateServiceTest(svc serviceTestSuite) (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("package %s\n\n", g.suite.packageName))

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"testing\"\n")

	if svc.servicePackage != "" {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("\tpb%s \"%s\"\n", toSnakeCase(svc.serviceName), svc.servicePackage))
	}

	buf.WriteString(")\n\n")

	// Generate test function for each method
	for _, method := range svc.methods {
		testFunc, err := g.generateMethodTest(svc, method)
		if err != nil {
			return "", fmt.Errorf("failed to generate test for %s: %w", method.methodName, err)
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
func (g *codeGenerator) generateMethodTest(svc serviceTestSuite, method methodTestSuite) (string, error) {
	if len(method.testCases) == 0 {
		return "", fmt.Errorf("no test cases for method %s", method.methodName)
	}

	firstCase := method.testCases[0]
	pbAlias := "pb" + toSnakeCase(svc.serviceName)

	var buf strings.Builder

	// Comment
	buf.WriteString(fmt.Sprintf("// Test%s tests the %s RPC call.\n", method.methodName, method.methodName))
	buf.WriteString("// This test was automatically generated from model checking execution.\n")

	// Function definition
	buf.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {\n", method.methodName))

	// Table definition
	buf.WriteString("\ttests := []struct {\n")
	buf.WriteString("\t\tname     string\n")
	buf.WriteString(fmt.Sprintf("\t\tinput    *%s.%s\n", pbAlias, firstCase.inputType))
	buf.WriteString(fmt.Sprintf("\t\texpected *%s.%s\n", pbAlias, firstCase.outputType))
	buf.WriteString("\t}{\n")

	// Each test case
	for _, tc := range method.testCases {
		buf.WriteString("\t\t{\n")
		buf.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", tc.name))
		buf.WriteString(fmt.Sprintf("\t\t\tinput: %s,\n", formatStructLiteral(pbAlias, tc.inputType, tc.input)))
		buf.WriteString(fmt.Sprintf("\t\t\texpected: %s,\n", formatStructLiteral(pbAlias, tc.outputType, tc.output)))
		buf.WriteString("\t\t},\n")
	}

	buf.WriteString("\t}\n\n")

	// Test loop
	buf.WriteString("\tfor _, tt := range tests {\n")
	buf.WriteString("\t\tt.Run(tt.name, func(t *testing.T) {\n")
	buf.WriteString("\t\t\tctx := context.Background()\n")
	buf.WriteString(fmt.Sprintf("\t\t\tactual, err := %s.%s(ctx, tt.input)\n",
		svc.clientVarName, method.methodName))
	buf.WriteString("\t\t\tif err != nil {\n")
	buf.WriteString("\t\t\t\tt.Fatalf(\"RPC call failed: %%v\", err)\n")
	buf.WriteString("\t\t\t}\n\n")
	buf.WriteString("\t\t\t// Verify the output matches expected\n")
	buf.WriteString("\t\t\tif !compareE2EOutput(tt.expected, actual) {\n")
	buf.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s output mismatch:\\nexpected: %%+v\\ngot:      %%+v\", tt.expected, actual)\n",
		method.methodName))
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t})\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	return buf.String(), nil
}

// formatStructLiteral formats a map as a Go struct literal.
// The 'any' parameter is necessary because field values can be of various types.
func formatStructLiteral(pkgAlias, typeName string, data map[string]any) string {
	if len(data) == 0 {
		return fmt.Sprintf("&%s.%s{}", pkgAlias, typeName)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	fmt.Fprintf(&buf, "&%s.%s{\n", pkgAlias, typeName)
	for _, k := range keys {
		fmt.Fprintf(&buf, "\t\t\t\t%s: %s,\n", k, formatValue(data[k]))
	}
	buf.WriteString("\t\t\t}")

	return buf.String()
}

// formatValue formats a value as a Go literal.
// The 'any' type is required here because protobuf field values can be of various types
// (string, bool, int64, []string, etc.) determined at runtime.
func formatValue(value any) string {
	if value == nil {
		return "nil"
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		if val.IsNil() {
			return "nil"
		}
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%#v", value)
	}
}
