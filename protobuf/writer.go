package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goatx/goat/internal/strcase"
)

type fileWriter struct {
	outputDir   string
	packageName string
	goPackage   string
}

func newFileWriter(outputDir, packageName, goPackage string) *fileWriter {
	return &fileWriter{
		outputDir:   outputDir,
		packageName: packageName,
		goPackage:   goPackage,
	}
}

func (w *fileWriter) writeProtoFile(filename string, definitions *definitions) error {
	if err := os.MkdirAll(w.outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	content := w.generateFileContent(definitions)

	filePath := filepath.Join(w.outputDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write proto file: %w", err)
	}

	return nil
}

func (w *fileWriter) generateFileContent(definitions *definitions) string {
	var builder strings.Builder

	builder.WriteString("syntax = \"proto3\";\n\n")

	if w.packageName != "" {
		builder.WriteString("package ")
		builder.WriteString(w.packageName)
		builder.WriteString(";\n")
	}

	if w.packageName != "" && w.goPackage != "" {
		builder.WriteString("\n")
	}

	if w.goPackage != "" {
		builder.WriteString("option go_package = \"")
		builder.WriteString(w.goPackage)
		builder.WriteString("\";\n")
	}

	if w.packageName != "" || w.goPackage != "" {
		builder.WriteString("\n")
	}

	for i, message := range definitions.Messages {
		w.writeMessage(&builder, message)
		builder.WriteString("\n")
		if i < len(definitions.Messages)-1 {
			builder.WriteString("\n")
		}
	}
	if len(definitions.Messages) > 0 && len(definitions.Services) > 0 {
		builder.WriteString("\n")
	}

	for _, service := range definitions.Services {
		w.writeService(&builder, service)
		builder.WriteString("\n")
	}

	return builder.String()
}

func (*fileWriter) writeMessage(builder *strings.Builder, message *message) {
	builder.WriteString("message ")
	builder.WriteString(message.Name)
	builder.WriteString(" {\n")

	for _, field := range message.Fields {
		fieldType := field.Type
		if field.IsRepeated {
			fieldType = "repeated " + fieldType
		}

		fieldName := strcase.ToSnakeCase(field.Name)
		builder.WriteString("  ")
		builder.WriteString(fieldType)
		builder.WriteString(" ")
		builder.WriteString(fieldName)
		builder.WriteString(" = ")
		builder.WriteString(strconv.Itoa(field.Number))
		builder.WriteString(";\n")
	}

	builder.WriteString("}")
}

func (*fileWriter) writeService(builder *strings.Builder, service *service) {
	builder.WriteString("service ")
	builder.WriteString(service.Name)
	builder.WriteString(" {\n")

	for _, method := range service.Methods {
		builder.WriteString("  rpc ")
		builder.WriteString(method.Name)
		builder.WriteString("(")
		builder.WriteString(method.InputType)
		builder.WriteString(") returns (")
		builder.WriteString(method.OutputType)
		builder.WriteString(");\n")
	}

	builder.WriteString("}")
}
