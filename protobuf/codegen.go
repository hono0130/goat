package protobuf

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

// GoTestGenerator generates Go test code from E2E test cases.
type GoTestGenerator struct {
	// PackageName is the package name for the generated test file
	PackageName string

	// Imports are additional imports to include (beyond the standard ones)
	Imports []string
}

// NewGoTestGenerator creates a new Go test code generator.
func NewGoTestGenerator(packageName string) *GoTestGenerator {
	return &GoTestGenerator{
		PackageName: packageName,
		Imports:     []string{},
	}
}

// AddImport adds an import to the generated test file.
func (g *GoTestGenerator) AddImport(importPath string) {
	g.Imports = append(g.Imports, importPath)
}

// GenerateMultiple generates Go test code from multiple E2E test cases.
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
	buf.WriteString("\t\"testing\"\n")
	buf.WriteString("\t\"reflect\"\n")

	// Add custom imports
	for _, imp := range g.Imports {
		buf.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}

	buf.WriteString(")\n\n")

	// Generate test functions for each test case
	for i, testCase := range testCases {
		testFunc, err := g.generateTestFunction(i, testCase)
		if err != nil {
			return "", fmt.Errorf("failed to generate test function %d: %w", i, err)
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

// generateTestFunction generates a single test function from an E2E test case.
func (g *GoTestGenerator) generateTestFunction(index int, testCase E2ETestCase) (string, error) {
	tmpl := `// Test{{.MethodName}}_{{.Index}} tests the {{.MethodName}} RPC call.
// This test was automatically generated from model checking execution.
func Test{{.MethodName}}_{{.Index}}(t *testing.T) {
	// Input: {{.InputType}}
	input := {{.InputValue}}

	// Expected output: {{.OutputType}}
	expected := {{.OutputValue}}

	// TODO: Execute the RPC call to get actual output
	// Example:
	//   service := &UserService{}
	//   ctx := context.Background()
	//   output := service.{{.MethodName}}(ctx, input)
	//
	// For now, this is a placeholder that will fail until you implement the execution.
	var output interface{}
	_ = input // Use input when implementing

	// Verify the output matches expected
	if !compareE2EOutput(expected, output) {
		t.Errorf("{{.MethodName}} output mismatch:\nexpected: %+v\ngot:      %+v", expected, output)
	}
}
`

	data := struct {
		Index       int
		MethodName  string
		InputType   string
		InputValue  string
		OutputType  string
		OutputValue string
	}{
		Index:       index,
		MethodName:  testCase.MethodName,
		InputType:   testCase.InputType,
		InputValue:  g.formatStructLiteral(testCase.InputType, testCase.Input),
		OutputType:  testCase.OutputType,
		OutputValue: g.formatStructLiteral(testCase.OutputType, testCase.Output),
	}

	t := template.Must(template.New("test").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// formatStructLiteral formats a map as a Go struct literal.
func (g *GoTestGenerator) formatStructLiteral(typeName string, data map[string]any) string {
	if len(data) == 0 {
		return fmt.Sprintf("&%s{}", typeName)
	}

	var fields []string
	for key, value := range data {
		fields = append(fields, fmt.Sprintf("\t\t%s: %s", key, g.formatValue(value)))
	}

	return fmt.Sprintf("&%s{\n%s,\n\t}", typeName, strings.Join(fields, ",\n"))
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

// toTestName converts a test case name to a valid Go test function name.
func toTestName(name string) string {
	// Replace non-alphanumeric characters with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)

	// Capitalize first letter
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}

	return result
}
