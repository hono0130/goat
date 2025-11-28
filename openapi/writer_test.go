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
		w := newSpecWriter(tempDir, "Test API", "1.0.0")

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
	w := &specWriter{
		title:   "Test API",
		version: "1.0.0",
	}

	got := w.generateFileContent(&openAPIDefinitions{
		Schemas: []*schemaDefinition{},
		Paths:   []*pathDefinition{},
	})

	want := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0

paths:

components:
  schemas:
`

	if got != want {
		t.Errorf("generateFileContent() = %q, want %q", got, want)
	}
}

func TestSpecWriter_writePath(t *testing.T) {
	t.Parallel()

	bodySchema := &schemaDefinition{
		Name: "CreateUserRequest",
		Fields: []schemaField{
			{Name: "UserID", Type: "string", ParamType: parameterTypeNone, Required: true},
		},
	}

	tests := []struct {
		name string
		path *pathDefinition
		want string
	}{
		{
			name: "writes request body as required by default",
			path: &pathDefinition{
				Path: "/users",
				Operations: []pathOperation{
					{
						Method:        HTTPMethodPost,
						OperationID:   "createUser",
						RequestRef:    "CreateUserRequest",
						RequestSchema: bodySchema,
						Responses: []operationResponse{
							{StatusCode: StatusOK, ResponseRef: "CreateUserResponse"},
						},
					},
				},
			},
			want: `  /users:
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
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateUserResponse'
`,
		},
		{
			name: "respects optional request body flag",
			path: &pathDefinition{
				Path: "/users",
				Operations: []pathOperation{
					{
						Method:         HTTPMethodPost,
						OperationID:    "createUser",
						RequestRef:     "CreateUserRequest",
						RequestSchema:  bodySchema,
						IsBodyOptional: true,
						Responses: []operationResponse{
							{StatusCode: StatusCreated, ResponseRef: "CreateUserResponse"},
						},
					},
				},
			},
			want: `  /users:
    post:
      operationId: createUser
      requestBody:
        required: false
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateUserResponse'
`,
		},
		{
			name: "omits request body for unsupported methods",
			path: &pathDefinition{
				Path: "/users",
				Operations: []pathOperation{
					{
						Method:        HTTPMethodGet,
						OperationID:   "getUser",
						RequestRef:    "GetUserRequest",
						RequestSchema: bodySchema,
						Responses: []operationResponse{
							{StatusCode: StatusOK, ResponseRef: "GetUserResponse"},
						},
					},
				},
			},
			want: `  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GetUserResponse'
`,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var b strings.Builder
			w := &specWriter{}
			w.writePath(&b, tt.path)

			if b.String() != tt.want {
				t.Errorf("writePath() = %q, want %q", b.String(), tt.want)
			}
		})
	}
}

func TestSpecWriter_writeSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema *schemaDefinition
		want   string
	}{
		{
			name: "includes only body fields and required list",
			schema: &schemaDefinition{
				Name: "Sample",
				Fields: []schemaField{
					{Name: "userID", Type: "string", Required: true},
					{Name: "tags", Type: "string", Format: "uuid", IsArray: true, Required: true},
					{Name: "id", Type: "string", ParamType: parameterTypePath},
				},
			},
			want: `    Sample:
      type: object
      properties:
        userID:
          type: string
        tags:
          type: array
          items:
            type: string
            format: uuid
      required:
        - userID
        - tags
`,
		},
		{
			name: "handles optional field with format",
			schema: &schemaDefinition{
				Name: "Optional",
				Fields: []schemaField{
					{Name: "traceID", Type: "string", Format: "uuid", Required: false},
				},
			},
			want: `    Optional:
      type: object
      properties:
        traceID:
          type: string
          format: uuid
`,
		},
		{
			name: "writes nothing when only parameters exist",
			schema: &schemaDefinition{
				Name: "ParamOnly",
				Fields: []schemaField{
					{Name: "query", Type: "string", ParamType: parameterTypeQuery},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var b strings.Builder
			w := &specWriter{}
			w.writeSchema(&b, tt.schema)

			if b.String() != tt.want {
				t.Errorf("writeSchema() = %q, want %q", b.String(), tt.want)
			}
		})
	}
}

func TestSpecWriter_writeParameters(t *testing.T) {
	t.Parallel()

	requestSchema := &schemaDefinition{
		Name: "Request",
		Fields: []schemaField{
			{Name: "id", Type: "string", ParamType: parameterTypePath},
			{Name: "page", Type: "integer", Format: "int32", ParamType: parameterTypeQuery},
			{Name: "X-Token", Type: "string", ParamType: parameterTypeHeader, Required: true},
		},
	}

	var b strings.Builder
	w := &specWriter{}
	w.writeParameters(&b, requestSchema)

	want := `      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: page
          in: query
          required: false
          schema:
            type: integer
            format: int32
        - name: X-Token
          in: header
          required: true
          schema:
            type: string
`

	if b.String() != want {
		t.Errorf("writeParameters() = %q, want %q", b.String(), want)
	}
}
