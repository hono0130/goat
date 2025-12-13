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

	if err := os.MkdirAll(opts.OutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	testSuite := &testSuite{suite: suite}

	mainCode, err := testSuite.generateMainTest()
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainPath, []byte(mainCode), 0o600); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	for _, group := range suite.Groups {
		serviceCode, err := testSuite.generateServiceTest(group)
		if err != nil {
			return fmt.Errorf("failed to generate test for %s: %w", group.Name, err)
		}

		filename := strcase.ToSnakeCase(group.Name) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(serviceCode), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}
