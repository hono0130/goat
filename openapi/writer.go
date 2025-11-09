package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type specWriter struct {
	outputDir   string
	title       string
	version     string
	description string
}

func newSpecWriter(outputDir, title, version, description string) *specWriter {
	return &specWriter{
		outputDir:   outputDir,
		title:       title,
		version:     version,
		description: description,
	}
}

func (w *specWriter) writeOpenAPIFile(filename string, definitions *openAPIDefinitions) error {
	if err := os.MkdirAll(w.outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	content := w.generateFileContent(definitions)

	filePath := filepath.Join(w.outputDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write openapi file: %w", err)
	}

	return nil
}

func (w *specWriter) generateFileContent(definitions *openAPIDefinitions) string {
	var builder strings.Builder

	builder.WriteString("openapi: 3.0.0\n")
	builder.WriteString("info:\n")
	if w.title != "" {
		builder.WriteString("  title: ")
		builder.WriteString(w.title)
		builder.WriteString("\n")
	}
	if w.description != "" {
		builder.WriteString("  description: ")
		builder.WriteString(w.description)
		builder.WriteString("\n")
	}
	builder.WriteString("  version: ")
	builder.WriteString(w.version)
	builder.WriteString("\n")

	builder.WriteString("\n")
	builder.WriteString("paths:\n")
	for _, path := range definitions.Paths {
		w.writePath(&builder, path)
	}

	builder.WriteString("\n")
	builder.WriteString("components:\n")
	builder.WriteString("  schemas:\n")
	for _, schema := range definitions.Schemas {
		w.writeSchema(&builder, schema)
	}

	return builder.String()
}

func (*specWriter) writePath(builder *strings.Builder, path *pathDefinition) {
	builder.WriteString("  ")
	builder.WriteString(path.Path)
	builder.WriteString(":\n")

	for _, operation := range path.Operations {
		methodLower := strings.ToLower(operation.Method)
		builder.WriteString("    ")
		builder.WriteString(methodLower)
		builder.WriteString(":\n")

		if operation.OperationID != "" {
			builder.WriteString("      operationId: ")
			builder.WriteString(operation.OperationID)
			builder.WriteString("\n")
		}

		builder.WriteString("      requestBody:\n")
		builder.WriteString("        required: true\n")
		builder.WriteString("        content:\n")
		builder.WriteString("          application/json:\n")
		builder.WriteString("            schema:\n")
		builder.WriteString("              $ref: '#/components/schemas/")
		builder.WriteString(operation.RequestRef)
		builder.WriteString("'\n")

		builder.WriteString("      responses:\n")
		builder.WriteString("        '200':\n")
		builder.WriteString("          description: Successful response\n")
		builder.WriteString("          content:\n")
		builder.WriteString("            application/json:\n")
		builder.WriteString("              schema:\n")
		builder.WriteString("                $ref: '#/components/schemas/")
		builder.WriteString(operation.ResponseRef)
		builder.WriteString("'\n")
	}
}

func (w *specWriter) writeSchema(builder *strings.Builder, schema *schemaDefinition) {
	builder.WriteString("    ")
	builder.WriteString(schema.Name)
	builder.WriteString(":\n")
	builder.WriteString("      type: object\n")

	if len(schema.Fields) > 0 {
		builder.WriteString("      properties:\n")
		for _, field := range schema.Fields {
			fieldName := w.toCamelCase(field.Name)
			builder.WriteString("        ")
			builder.WriteString(fieldName)
			builder.WriteString(":\n")

			if field.IsArray {
				builder.WriteString("          type: array\n")
				builder.WriteString("          items:\n")
				builder.WriteString("            type: ")
				builder.WriteString(field.Type)
				builder.WriteString("\n")
				if field.Format != "" {
					builder.WriteString("            format: ")
					builder.WriteString(field.Format)
					builder.WriteString("\n")
				}
			} else {
				builder.WriteString("          type: ")
				builder.WriteString(field.Type)
				builder.WriteString("\n")
				if field.Format != "" {
					builder.WriteString("          format: ")
					builder.WriteString(field.Format)
					builder.WriteString("\n")
				}
			}
		}

		requiredFields := []string{}
		for _, field := range schema.Fields {
			if field.Required {
				requiredFields = append(requiredFields, w.toCamelCase(field.Name))
			}
		}

		if len(requiredFields) > 0 {
			builder.WriteString("      required:\n")
			for _, fieldName := range requiredFields {
				builder.WriteString("        - ")
				builder.WriteString(fieldName)
				builder.WriteString("\n")
			}
		}
	}
}

func (*specWriter) toCamelCase(name string) string {
	if name == "" {
		return name
	}

	firstChar := rune(name[0])
	if firstChar >= 'A' && firstChar <= 'Z' {
		return string(firstChar+32) + name[1:]
	}

	return name
}
