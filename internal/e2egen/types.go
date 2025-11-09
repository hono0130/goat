package e2egen

// TestSuite represents the entire test suite to be generated.
// This is a protocol-agnostic intermediate representation that can be used
// for both protobuf and OpenAPI E2E test generation.
type TestSuite struct {
	PackageName string
	Services    []ServiceTestSuite
}

// ServiceTestSuite represents test cases for a single service.
type ServiceTestSuite struct {
	ServiceName    string // e.g., "UserService"
	ServicePackage string // import path (e.g., "github.com/example/proto/user")
	ClientVarName  string // e.g., "user_serviceClient"
	Methods        []MethodTestSuite
}

// MethodTestSuite represents test cases for a single RPC method or API endpoint.
type MethodTestSuite struct {
	MethodName string // e.g., "CreateUser"
	TestCases  []TestCase
}

// TestCase represents a single test case with input and expected output.
type TestCase struct {
	Name       string         // e.g., "case_0"
	InputType  string         // e.g., "CreateUserRequest"
	Input      map[string]any // serialized input data (any is necessary: field values can be string, bool, int64, []any, etc.)
	OutputType string         // e.g., "CreateUserResponse"
	Output     map[string]any // serialized output data (any is necessary: field values can be string, bool, int64, []any, etc.)
}
