package openapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateOpenAPI_GoldenFile(t *testing.T) {
	tests := []struct {
		name       string
		setupSpec  func() *OpenAPIServiceSpec[*TestService1]
		opts       GenerateOptions
		wantGolden string
	}{
		{
			name: "generates complete OpenAPI spec with multiple endpoints",
			setupSpec: func() *OpenAPIServiceSpec[*TestService1] {
				spec := NewOpenAPIServiceSpec(&TestService1{})
				state := &TestIdleState{}
				spec.DefineStates(state).SetInitialState(state)

				OnOpenAPIRequest[*TestService1, *TestRequest1, *TestResponse1](spec, state, "POST", "/items", "createItem",
					func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
						return OpenAPISendTo(ctx, sm, &TestResponse1{})
					})

				OnOpenAPIRequest[*TestService1, *TestRequest2, *TestResponse2](spec, state, "GET", "/items/{id}", "getItem",
					func(ctx context.Context, event *TestRequest2, sm *TestService1) OpenAPIResponse[*TestResponse2] {
						return OpenAPISendTo(ctx, sm, &TestResponse2{})
					})

				return spec
			},
			opts: GenerateOptions{
				Title:       "Test API",
				Version:     "1.0.0",
				Description: "Test API Description",
				Filename:    "test.yaml",
			},
			wantGolden: `openapi: 3.0.0
info:
  title: Test API
  description: Test API Description
  version: 1.0.0

paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestRequest1'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestResponse1'
  /items/{id}:
    get:
      operationId: getItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestRequest2'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestResponse2'

components:
  schemas:
    TestRequest1:
      type: object
      properties:
        data:
          type: string
      required:
        - data
    TestRequest2:
      type: object
      properties:
        info:
          type: string
      required:
        - info
    TestResponse1:
      type: object
      properties:
        result:
          type: string
      required:
        - result
    TestResponse2:
      type: object
      properties:
        value:
          type: string
      required:
        - value
`,
		},
		{
			name: "generates minimal OpenAPI spec",
			setupSpec: func() *OpenAPIServiceSpec[*TestService1] {
				spec := NewOpenAPIServiceSpec(&TestService1{})
				state := &TestIdleState{}
				spec.DefineStates(state).SetInitialState(state)

				OnOpenAPIRequest[*TestService1, *TestRequest1, *TestResponse1](spec, state, "GET", "/health", "healthCheck",
					func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
						return OpenAPISendTo(ctx, sm, &TestResponse1{})
					})

				return spec
			},
			opts: GenerateOptions{
				Title:    "Health API",
				Version:  "0.1.0",
				Filename: "health.yaml",
			},
			wantGolden: `openapi: 3.0.0
info:
  title: Health API
  version: 0.1.0

paths:
  /health:
    get:
      operationId: healthCheck
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestRequest1'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestResponse1'

components:
  schemas:
    TestRequest1:
      type: object
      properties:
        data:
          type: string
      required:
        - data
    TestResponse1:
      type: object
      properties:
        result:
          type: string
      required:
        - result
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := tt.setupSpec()
			tempDir := t.TempDir()
			tt.opts.OutputDir = tempDir

			err := GenerateOpenAPI(&tt.opts, spec)
			if err != nil {
				t.Fatalf("GenerateOpenAPI() error = %v", err)
			}

			generatedFile := filepath.Join(tempDir, tt.opts.Filename)
			// #nosec G304 - generatedFile is constructed from t.TempDir() and test options
			content, err := os.ReadFile(generatedFile)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			got := string(content)
			if got != tt.wantGolden {
				t.Errorf("Generated content mismatch.\nGot:\n%s\nWant:\n%s", got, tt.wantGolden)
			}
		})
	}
}

func TestGenerateOpenAPI_MultipleServices(t *testing.T) {
	spec1 := NewOpenAPIServiceSpec(&TestService1{})
	state1 := &TestIdleState{}
	spec1.DefineStates(state1).SetInitialState(state1)

	OnOpenAPIRequest[*TestService1, *TestRequest1, *TestResponse1](spec1, state1, "POST", "/service1/action", "service1Action",
		func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
			return OpenAPISendTo(ctx, sm, &TestResponse1{})
		})

	spec2 := NewOpenAPIServiceSpec(&TestService2{})
	state2 := &TestIdleState{}
	spec2.DefineStates(state2).SetInitialState(state2)

	OnOpenAPIRequest[*TestService2, *TestRequest2, *TestResponse2](spec2, state2, "GET", "/service2/query", "service2Query",
		func(ctx context.Context, event *TestRequest2, sm *TestService2) OpenAPIResponse[*TestResponse2] {
			return OpenAPISendTo(ctx, sm, &TestResponse2{})
		})

	tempDir := t.TempDir()
	opts := GenerateOptions{
		OutputDir:   tempDir,
		Title:       "Multi-Service API",
		Version:     "2.0.0",
		Description: "Combined API for multiple services",
		Filename:    "multi-service.yaml",
	}

	err := GenerateOpenAPI(&opts, spec1, spec2)
	if err != nil {
		t.Fatalf("GenerateOpenAPI() error = %v", err)
	}

	generatedFile := filepath.Join(tempDir, opts.Filename)
	// #nosec G304 - generatedFile is constructed from t.TempDir() and fixed filename
	content, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)
	want := `openapi: 3.0.0
info:
  title: Multi-Service API
  description: Combined API for multiple services
  version: 2.0.0

paths:
  /service1/action:
    post:
      operationId: service1Action
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestRequest1'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestResponse1'
  /service2/query:
    get:
      operationId: service2Query
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestRequest2'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TestResponse2'

components:
  schemas:
    TestRequest1:
      type: object
      properties:
        data:
          type: string
      required:
        - data
    TestResponse1:
      type: object
      properties:
        result:
          type: string
      required:
        - result
    TestRequest2:
      type: object
      properties:
        info:
          type: string
      required:
        - info
    TestResponse2:
      type: object
      properties:
        value:
          type: string
      required:
        - value
`

	if got != want {
		t.Errorf("Generated content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
