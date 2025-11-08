package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecWriter_writeOpenAPIFile(t *testing.T) {
	tempDir := t.TempDir()
	filename := "empty.yaml"
	t.Run("writes empty definitions", func(t *testing.T) {
		w := newSpecWriter(tempDir, "Test API", "1.0.0", "Test Description")

		err := w.writeOpenAPIFile(filename, &openAPIDefinitions{
			Schemas: []*schemaDefinition{},
			Paths:   []*pathDefinition{},
		})
		if err != nil {
			t.Errorf("writeOpenAPIFile() error = %v", err)
		}

		filePath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected file %s to be created", filePath)
		}

		// #nosec G304 - filePath is built from t.TempDir() and a fixed filename in test
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}

		if !strings.Contains(string(content), "openapi: 3.0.0") {
			t.Error("generated file missing openapi version declaration")
		}
	})
}

func TestSpecWriter_generateFileContent(t *testing.T) {
	tests := []struct {
		name        string
		writer      *specWriter
		definitions *openAPIDefinitions
		wantContent string
	}{
		{
			name: "generates minimal content",
			writer: &specWriter{
				title:       "Test API",
				version:     "1.0.0",
				description: "Test Description",
			},
			definitions: &openAPIDefinitions{
				Schemas: []*schemaDefinition{},
				Paths:   []*pathDefinition{},
			},
			wantContent: `openapi: 3.0.0
info:
  title: Test API
  description: Test Description
  version: 1.0.0

paths:

components:
  schemas:
`,
		},
		{
			name: "generates path with operation",
			writer: &specWriter{
				title:       "Test API",
				version:     "1.0.0",
				description: "Test Description",
			},
			definitions: &openAPIDefinitions{
				Schemas: []*schemaDefinition{},
				Paths: []*pathDefinition{
					{
						Path: "/users",
						Operations: []pathOperation{
							{
								Method:      "POST",
								OperationID: "createUser",
								RequestRef:  "CreateUserRequest",
								ResponseRef: "CreateUserResponse",
							},
						},
					},
				},
			},
			wantContent: `openapi: 3.0.0
info:
  title: Test API
  description: Test Description
  version: 1.0.0

paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateUserResponse'

components:
  schemas:
`,
		},
		{
			name: "generates schema with fields",
			writer: &specWriter{
				title:       "Test API",
				version:     "1.0.0",
				description: "Test Description",
			},
			definitions: &openAPIDefinitions{
				Schemas: []*schemaDefinition{
					{
						Name: "User",
						Fields: []schemaField{
							{
								Name:     "UserID",
								Type:     "string",
								Format:   "",
								IsArray:  false,
								Required: true,
							},
							{
								Name:     "Username",
								Type:     "string",
								Format:   "",
								IsArray:  false,
								Required: true,
							},
							{
								Name:     "Tags",
								Type:     "string",
								Format:   "",
								IsArray:  true,
								Required: true,
							},
						},
					},
				},
				Paths: []*pathDefinition{},
			},
			wantContent: `openapi: 3.0.0
info:
  title: Test API
  description: Test Description
  version: 1.0.0

paths:

components:
  schemas:
    User:
      type: object
      properties:
        userID:
          type: string
        username:
          type: string
        tags:
          type: array
          items:
            type: string
      required:
        - userID
        - username
        - tags
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.writer.generateFileContent(tt.definitions)

			if got != tt.wantContent {
				t.Errorf("generateFileContent() = %q, want %q", got, tt.wantContent)
			}
		})
	}
}

func TestSpecWriter_toCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "converts PascalCase to camelCase",
			input: "PascalCase",
			want:  "pascalCase",
		},
		{
			name:  "converts UserID to userID",
			input: "UserID",
			want:  "userID",
		},
		{
			name:  "handles single word",
			input: "Word",
			want:  "word",
		},
		{
			name:  "handles lowercase",
			input: "lowercase",
			want:  "lowercase",
		},
		{
			name:  "handles empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &specWriter{}
			got := w.toCamelCase(tt.input)

			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
