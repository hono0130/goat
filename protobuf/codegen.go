package protobuf

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"
)

// GoTestGenerator generates Go test code from E2E test cases.
type GoTestGenerator struct {
	// PackageName is the package name for the generated test file
	PackageName string

	// ServiceName is the gRPC service name (e.g., "UserService")
	ServiceName string

	// ServicePackage is the import path for the generated protobuf package
	ServicePackage string
}

// NewGoTestGenerator creates a new Go test code generator.
func NewGoTestGenerator(packageName string) *GoTestGenerator {
	return &GoTestGenerator{
		PackageName: packageName,
	}
}

// GenerateMultiple generates Go test code from multiple E2E test cases.
// Test cases with the same MethodName are grouped into a single table-driven test.
//
// Example:
//
//	generator := protobuf.NewGoTestGenerator("main")
//	code, err := generator.GenerateMultiple(testCases)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("user_service_test.go", []byte(code), 0644)
func (g *GoTestGenerator) GenerateMultiple(testCases []E2ETestCase) (string, error) {
	if len(testCases) == 0 {
		return "", fmt.Errorf("no test cases provided")
	}

	var buf bytes.Buffer

	// Generate package and imports
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")

	// Add required imports for gRPC testing
	if g.ServicePackage != "" {
		buf.WriteString("\t\"context\"\n")
		buf.WriteString("\t\"log\"\n")
		buf.WriteString("\t\"net\"\n")
		buf.WriteString("\t\"os\"\n")
	}
	buf.WriteString("\t\"reflect\"\n")
	buf.WriteString("\t\"testing\"\n")

	if g.ServicePackage != "" {
		buf.WriteString("\n")
		buf.WriteString("\t\"google.golang.org/grpc\"\n")
		buf.WriteString(fmt.Sprintf("\tpb \"%s\"\n", g.ServicePackage))
	}

	buf.WriteString(")\n\n")

	// Generate global client variable
	if g.ServicePackage != "" && g.ServiceName != "" {
		buf.WriteString(fmt.Sprintf("var client pb.%sClient\n\n", g.ServiceName))
	}

	// Generate TestMain if service is configured
	if g.ServicePackage != "" && g.ServiceName != "" {
		testMain := g.generateTestMain()
		buf.WriteString(testMain)
		buf.WriteString("\n\n")
	}

	// Group test cases by method name
	grouped := g.groupByMethod(testCases)

	// Get sorted method names for stable output
	methodNames := make([]string, 0, len(grouped))
	for methodName := range grouped {
		methodNames = append(methodNames, methodName)
	}
	sort.Strings(methodNames)

	// Generate table-driven test for each method
	for _, methodName := range methodNames {
		cases := grouped[methodName]
		testFunc, err := g.generateTableTest(methodName, cases)
		if err != nil {
			return "", fmt.Errorf("failed to generate test for %s: %w", methodName, err)
		}
		buf.WriteString(testFunc)
		buf.WriteString("\n\n")
	}

	// Generate helper function for output comparison
	buf.WriteString(g.generateCompareHelper())

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, return unformatted code for debugging
		return buf.String(), fmt.Errorf("failed to format generated code: %w", err)
	}

	return string(formatted), nil
}

// groupByMethod groups test cases by their method name.
func (g *GoTestGenerator) groupByMethod(testCases []E2ETestCase) map[string][]E2ETestCase {
	grouped := make(map[string][]E2ETestCase)
	for _, tc := range testCases {
		grouped[tc.MethodName] = append(grouped[tc.MethodName], tc)
	}
	return grouped
}

// generateTableTest generates a table-driven test for a specific method.
func (g *GoTestGenerator) generateTableTest(methodName string, testCases []E2ETestCase) (string, error) {
	if len(testCases) == 0 {
		return "", fmt.Errorf("no test cases for method %s", methodName)
	}

	// Get input and output types from first test case
	inputType := testCases[0].InputType
	outputType := testCases[0].OutputType

	// Check if client is configured
	useClient := g.ServicePackage != "" && g.ServiceName != ""

	var tmpl string
	if useClient {
		tmpl = `// Test{{.MethodName}} tests the {{.MethodName}} RPC call.
// This test was automatically generated from model checking execution.
func Test{{.MethodName}}(t *testing.T) {
	tests := []struct {
		name     string
		input    *pb.{{.InputType}}
		expected *pb.{{.OutputType}}
	}{
{{range .Cases}}		{
			name: "{{.Name}}",
			input: {{.InputValue}},
			expected: {{.OutputValue}},
		},
{{end}}	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, err := client.{{.MethodName}}(ctx, tt.input)
			if err != nil {
				t.Fatalf("RPC call failed: %v", err)
			}

			// Verify the output matches expected
			if !compareE2EOutput(tt.expected, actual) {
				t.Errorf("{{.MethodName}} output mismatch:\nexpected: %+v\ngot:      %+v", tt.expected, actual)
			}
		})
	}
}
`
	} else {
		tmpl = `// Test{{.MethodName}} tests the {{.MethodName}} RPC call.
// This test was automatically generated from model checking execution.
func Test{{.MethodName}}(t *testing.T) {
	tests := []struct {
		name     string
		input    *{{.InputType}}
		expected *{{.OutputType}}
	}{
{{range .Cases}}		{
			name: "{{.Name}}",
			input: {{.InputValue}},
			expected: {{.OutputValue}},
		},
{{end}}	}

	// TODO: Setup your gRPC client and implement RPC calls
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Execute the RPC call
			// ctx := context.Background()
			// actual, err := client.{{.MethodName}}(ctx, tt.input)
			// if err != nil {
			//     t.Fatalf("RPC call failed: %v", err)
			// }

			// Placeholder - replace with actual RPC call above
			var actual *{{.OutputType}}
			_ = tt.input // Remove this line after implementing RPC call

			// Verify the output matches expected
			if !compareE2EOutput(tt.expected, actual) {
				t.Errorf("{{.MethodName}} output mismatch:\nexpected: %+v\ngot:      %+v", tt.expected, actual)
			}
		})
	}
}
`
	}

	type caseData struct {
		Name        string
		InputValue  string
		OutputValue string
	}

	cases := make([]caseData, len(testCases))
	for i, tc := range testCases {
		cases[i] = caseData{
			Name:        fmt.Sprintf("case_%d", i),
			InputValue:  g.formatStructLiteral(tc.InputType, tc.Input),
			OutputValue: g.formatStructLiteral(tc.OutputType, tc.Output),
		}
	}

	data := struct {
		MethodName string
		InputType  string
		OutputType string
		Cases      []caseData
	}{
		MethodName: methodName,
		InputType:  inputType,
		OutputType: outputType,
		Cases:      cases,
	}

	t := template.Must(template.New("tabletest").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// formatStructLiteral formats a map as a Go struct literal.
func (g *GoTestGenerator) formatStructLiteral(typeName string, data map[string]any) string {
	// Add pb. prefix if using protobuf package
	prefix := ""
	if g.ServicePackage != "" {
		prefix = "pb."
	}

	if len(data) == 0 {
		return fmt.Sprintf("&%s%s{}", prefix, typeName)
	}

	// Sort keys for stable output
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var fields []string
	for _, key := range keys {
		fields = append(fields, fmt.Sprintf("\t\t%s: %s", key, g.formatValue(data[key])))
	}

	return fmt.Sprintf("&%s%s{\n%s,\n\t}", prefix, typeName, strings.Join(fields, ",\n"))
}

// formatValue formats a value as a Go literal.
func (g *GoTestGenerator) formatValue(value any) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	case []interface{}:
		if len(v) == 0 {
			return "nil"
		}
		var items []string
		for _, item := range v {
			items = append(items, g.formatValue(item))
		}
		return fmt.Sprintf("[]string{%s}", strings.Join(items, ", "))
	default:
		return fmt.Sprintf("%#v", v)
	}
}

// generateCompareHelper generates a helper function for comparing outputs.
func (g *GoTestGenerator) generateCompareHelper() string {
	return `// compareE2EOutput compares two values for equality in E2E tests.
// This is a helper function automatically generated for E2E testing.
func compareE2EOutput(expected, actual interface{}) bool {
	return reflect.DeepEqual(expected, actual)
}
`
}

// generateTestMain generates the TestMain function for gRPC server setup.
func (g *GoTestGenerator) generateTestMain() string {
	tmpl := `// TestMain sets up the gRPC server and client for testing.
func TestMain(m *testing.M) {
	// Start gRPC server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// TODO: Register your service implementation here
	// pb.Register{{.ServiceName}}Server(grpcServer, &yourServiceImplementation{})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Create client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	client = pb.New{{.ServiceName}}Client(conn)

	// Run tests
	code := m.Run()

	// Cleanup
	conn.Close()
	grpcServer.Stop()

	os.Exit(code)
}
`

	data := struct {
		ServiceName string
	}{
		ServiceName: g.ServiceName,
	}

	t := template.Must(template.New("testmain").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return ""
	}

	return buf.String()
}
