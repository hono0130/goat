package protobuf

// testSuite represents the entire test suite to be generated.
// This is a protocol-agnostic intermediate representation.
// All fields are unexported as this is an internal implementation detail.
type testSuite struct {
	packageName string
	services    []serviceTestSuite
}

// serviceTestSuite represents test cases for a single service.
type serviceTestSuite struct {
	serviceName    string // e.g., "UserService"
	servicePackage string // import path (e.g., "github.com/example/proto/user")
	clientVarName  string // e.g., "user_serviceClient"
	methods        []methodTestSuite
}

// methodTestSuite represents test cases for a single RPC method.
type methodTestSuite struct {
	methodName string // e.g., "CreateUser"
	testCases  []testCase
}

// testCase represents a single test case with input and expected output.
type testCase struct {
	name       string         // e.g., "case_0"
	inputType  string         // e.g., "CreateUserRequest"
	input      map[string]any // serialized input data (any is necessary: field values can be string, bool, int64, []any, etc.)
	outputType string         // e.g., "CreateUserResponse"
	output     map[string]any // serialized output data (any is necessary: field values can be string, bool, int64, []any, etc.)
}
