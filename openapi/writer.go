package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type specWriter struct {
	outputDir string
	title     string
	version   string
}

func newSpecWriter(outputDir, title, version string) *specWriter {
	return &specWriter{
		outputDir: outputDir,
		title:     title,
		version:   version,
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
	builder.WriteString("  title: ")
	builder.WriteString(w.title)
	builder.WriteString("\n")
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

func (w *specWriter) writePath(builder *strings.Builder, path *pathDefinition) {
	builder.WriteString("  ")
	builder.WriteString(path.Path)
	builder.WriteString(":\n")

	for _, operation := range path.Operations {
		methodLower := strings.ToLower(operation.Method.String())
		builder.WriteString("    ")
		builder.WriteString(methodLower)
		builder.WriteString(":\n")

		if operation.OperationID != "" {
			builder.WriteString("      operationId: ")
			builder.WriteString(operation.OperationID)
			builder.WriteString("\n")
		}

		w.writeParameters(builder, operation.RequestSchema)

		bodyFields := make([]schemaField, 0)
		for _, field := range operation.RequestSchema.Fields {
			if field.ParamType == parameterTypeNone {
				bodyFields = append(bodyFields, field)
			}
		}

		if len(bodyFields) > 0 && operation.Method == HTTPMethodPost || operation.Method == HTTPMethodPut {
			builder.WriteString("      requestBody:\n")
			builder.WriteString("        required: ")
			if operation.IsBodyOptional {
				builder.WriteString("false\n")
			} else {
				builder.WriteString("true\n")
			}
			builder.WriteString("        content:\n")
			builder.WriteString("          application/json:\n")
			builder.WriteString("            schema:\n")
			builder.WriteString("              $ref: '#/components/schemas/")
			builder.WriteString(operation.RequestRef)
			builder.WriteString("'\n")
		}

		builder.WriteString("      responses:\n")
		for _, response := range operation.Responses {
			builder.WriteString("        '")
			fmt.Fprintf(builder, "%d", response.StatusCode)
			builder.WriteString("':\n")
			builder.WriteString("          description: ")
			builder.WriteString(descriptionForStatus(response.StatusCode))
			builder.WriteString("\n")
			builder.WriteString("          content:\n")
			builder.WriteString("            application/json:\n")
			builder.WriteString("              schema:\n")
			builder.WriteString("                $ref: '#/components/schemas/")
			builder.WriteString(response.ResponseRef)
			builder.WriteString("'\n")
		}
	}
}

func (*specWriter) writeSchema(builder *strings.Builder, schema *schemaDefinition) {
	bodyFields := make([]schemaField, 0)
	for _, field := range schema.Fields {
		if field.ParamType == parameterTypeNone {
			bodyFields = append(bodyFields, field)
		}
	}

	if len(bodyFields) == 0 {
		return
	}

	builder.WriteString("    ")
	builder.WriteString(schema.Name)
	builder.WriteString(":\n")
	builder.WriteString("      type: object\n")
	builder.WriteString("      properties:\n")
	for _, field := range bodyFields {
		builder.WriteString("        ")
		builder.WriteString(field.Name)
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

	requiredFields := make([]string, 0)
	for _, field := range bodyFields {
		if field.Required {
			requiredFields = append(requiredFields, field.Name)
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

func (*specWriter) writeParameters(builder *strings.Builder, requestSchema *schemaDefinition) {
	if requestSchema == nil {
		return
	}

	wroteParameters := false
	for _, field := range requestSchema.Fields {
		if field.ParamType == parameterTypeNone {
			continue
		}

		if !wroteParameters {
			builder.WriteString("      parameters:\n")
			wroteParameters = true
		}

		required := field.Required
		if field.ParamType == parameterTypePath {
			required = true
		}

		builder.WriteString("        - name: ")
		builder.WriteString(field.Name)
		builder.WriteString("\n")
		builder.WriteString("          in: ")
		builder.WriteString(field.ParamType.String())
		builder.WriteString("\n")
		builder.WriteString("          required: ")
		if required {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
		builder.WriteString("\n")
		builder.WriteString("          schema:\n")
		builder.WriteString("            type: ")
		builder.WriteString(field.Type)
		builder.WriteString("\n")
		if field.Format != "" {
			builder.WriteString("            format: ")
			builder.WriteString(field.Format)
			builder.WriteString("\n")
		}
	}
}

func descriptionForStatus(code StatusCode) string {
	if desc, ok := statusDescriptions[code]; ok {
		return desc
	}

	switch {
	case code >= 100 && code < 200:
		return "Informational response"
	case code >= 200 && code < 300:
		return "Successful response"
	case code >= 300 && code < 400:
		return "Redirect response"
	case code >= 400 && code < 500:
		return "Client error response"
	case code >= 500 && code < 600:
		return "Server error response"
	default:
		return "Response"
	}
}

var statusDescriptions = map[StatusCode]string{
	StatusOK:                  "OK",
	StatusCreated:             "Created",
	StatusAccepted:            "Accepted",
	StatusNoContent:           "No Content",
	StatusBadRequest:          "Bad Request",
	StatusUnauthorized:        "Unauthorized",
	StatusForbidden:           "Forbidden",
	StatusNotFound:            "Not Found",
	StatusConflict:            "Conflict",
	StatusInternalServerError: "Internal Server Error",
	StatusServiceUnavailable:  "Service Unavailable",
}
