package protobuf

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/goatx/goat/internal/e2egen"
	"github.com/goatx/goat/internal/strcase"
)

type codeGenerator struct {
	suite e2egen.TestSuite
}

func (g *codeGenerator) generateMainTest() (string, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n\n", g.suite.PackageName)

	b.WriteString("import (\n\t\"context\"\n\t\"log\"\n\t\"net\"\n\t\"os\"\n\t\"reflect\"\n\t\"testing\"\n\n\t\"google.golang.org/grpc\"\n")
	seen := make(map[string]bool)
	for _, svc := range g.suite.Services {
		if svc.ServicePackage != "" && !seen[svc.ServicePackage] {
			fmt.Fprintf(&b, "\tpb%s \"%s\"\n", strcase.ToSnakeCase(svc.ServiceName), svc.ServicePackage)
			seen[svc.ServicePackage] = true
		}
	}
	b.WriteString(")\n\n")

	for _, svc := range g.suite.Services {
		if svc.ServicePackage != "" {
			fmt.Fprintf(&b, "var %s pb%s.%sClient\n", svc.ClientVarName, strcase.ToSnakeCase(svc.ServiceName), svc.ServiceName)
		}
	}

	b.WriteString("\nfunc TestMain(m *testing.M) {\n")
	for i, svc := range g.suite.Services {
		if svc.ServicePackage == "" {
			continue
		}
		snake := strcase.ToSnakeCase(svc.ServiceName)
		fmt.Fprintf(&b, "\tlis%d, err := net.Listen(\"tcp\", \"localhost:0\")\n", i)
		b.WriteString("\tif err != nil {\n\t\tlog.Fatalf(\"Failed to listen: %%v\", err)\n\t}\n\n")
		fmt.Fprintf(&b, "\tgrpcServer%d := grpc.NewServer()\n", i)
		fmt.Fprintf(&b, "\t// TODO: Register your service implementation here\n")
		fmt.Fprintf(&b, "\t// pb%s.Register%sServer(grpcServer%d, &yourServiceImplementation{})\n\n", snake, svc.ServiceName, i)
		fmt.Fprintf(&b, "\tgo func() {\n\t\tif err := grpcServer%d.Serve(lis%d); err != nil {\n\t\t\tlog.Fatalf(\"Failed to serve: %%v\", err)\n\t\t}\n\t}()\n\n", i, i)
		fmt.Fprintf(&b, "\tconn%d, err := grpc.Dial(lis%d.Addr().String(), grpc.WithInsecure())\n", i, i)
		b.WriteString("\tif err != nil {\n\t\tlog.Fatalf(\"Failed to dial: %%v\", err)\n\t}\n")
		fmt.Fprintf(&b, "\t%s = pb%s.New%sClient(conn%d)\n\n", svc.ClientVarName, snake, svc.ServiceName, i)
	}

	b.WriteString("\tcode := m.Run()\n\n")
	for i, svc := range g.suite.Services {
		if svc.ServicePackage == "" {
			continue
		}
		fmt.Fprintf(&b, "\tconn%d.Close()\n\tgrpcServer%d.Stop()\n", i, i)
	}
	b.WriteString("\n\tos.Exit(code)\n}\n\n")
	b.WriteString("func compareE2EOutput(expected, actual any) bool {\n\treturn reflect.DeepEqual(expected, actual)\n}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return b.String(), fmt.Errorf("failed to format: %w", err)
	}
	return string(formatted), nil
}

func (g *codeGenerator) generateServiceTest(svc e2egen.ServiceTestSuite) (string, error) {
	var b strings.Builder
	snake := strcase.ToSnakeCase(svc.ServiceName)

	fmt.Fprintf(&b, "package %s\n\nimport (\n\t\"context\"\n\t\"testing\"\n", g.suite.PackageName)
	if svc.ServicePackage != "" {
		fmt.Fprintf(&b, "\n\tpb%s \"%s\"\n", snake, svc.ServicePackage)
	}
	b.WriteString(")\n\n")

	for _, method := range svc.Methods {
		if err := g.writeMethodTest(&b, svc, method); err != nil {
			return "", err
		}
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return b.String(), fmt.Errorf("failed to format: %w", err)
	}
	return string(formatted), nil
}

func (g *codeGenerator) writeMethodTest(b *strings.Builder, svc e2egen.ServiceTestSuite, method e2egen.MethodTestSuite) error {
	if len(method.TestCases) == 0 {
		return fmt.Errorf("no test cases for method %s", method.MethodName)
	}

	first := method.TestCases[0]
	alias := "pb" + strcase.ToSnakeCase(svc.ServiceName)

	fmt.Fprintf(b, "func Test%s(t *testing.T) {\n", method.MethodName)
	fmt.Fprintf(b, "\ttests := []struct {\n\t\tname     string\n\t\tinput    *%s.%s\n\t\texpected *%s.%s\n\t}{\n", alias, first.InputType, alias, first.OutputType)

	for _, tc := range method.TestCases {
		fmt.Fprintf(b, "\t\t{name: %q, input: %s, expected: %s},\n",
			tc.Name,
			e2egen.FormatStructLiteral(alias, tc.InputType, tc.Input),
			e2egen.FormatStructLiteral(alias, tc.OutputType, tc.Output))
	}

	b.WriteString("\t}\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n")
	fmt.Fprintf(b, "\t\t\tactual, err := %s.%s(context.Background(), tt.input)\n", svc.ClientVarName, method.MethodName)
	b.WriteString("\t\t\tif err != nil {\n\t\t\t\tt.Fatalf(\"RPC call failed: %%v\", err)\n\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\tif !compareE2EOutput(tt.expected, actual) {\n\t\t\t\tt.Errorf(\"%s mismatch:\\nexpected: %%+v\\ngot:      %%+v\", tt.expected, actual)\n\t\t\t}\n", method.MethodName)
	b.WriteString("\t\t})\n\t}\n}\n\n")

	return nil
}
