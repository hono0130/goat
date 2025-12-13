package e2egen

type TestSuite struct {
	PackageName string
	Services    []ServiceTestSuite
}

type ServiceTestSuite struct {
	ServiceName    string
	ServicePackage string
	ClientVarName  string
	Methods        []MethodTestSuite
}

type MethodTestSuite struct {
	MethodName string
	TestCases  []TestCase
}

type TestCase struct {
	Name       string
	InputType  string
	Input      map[string]any
	OutputType string
	Output     map[string]any
}
