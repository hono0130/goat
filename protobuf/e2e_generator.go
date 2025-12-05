package protobuf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goatx/goat/internal/strcase"
)

type MethodTestCase struct {
	MethodName string
	Inputs     []AbstractMessage
}

type ServiceTestCase struct {
	Spec           AbstractServiceSpec
	ServicePackage string
	Methods        []MethodTestCase
}

type E2ETestOptions struct {
	OutputDir   string
	PackageName string
	Services    []ServiceTestCase
}

func GenerateE2ETest(opts E2ETestOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./tests"
	}
	if opts.PackageName == "" {
		opts.PackageName = "main"
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	gen := &codeGenerator{suite: suite}

	mainCode, err := gen.generateMainTest()
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainPath, []byte(mainCode), 0644); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	for _, svc := range suite.Services {
		serviceCode, err := gen.generateServiceTest(svc)
		if err != nil {
			return fmt.Errorf("failed to generate test for %s: %w", svc.ServiceName, err)
		}

		filename := strcase.ToSnakeCase(svc.ServiceName) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(serviceCode), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}
