package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat/openapi"
)

func TestUserServiceOpenAPIGeneration(t *testing.T) {
	spec := createUserServiceModel()
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "openapi")

	opts := openapi.GenerateOptions{
		OutputDir: outputPath,
		Title:     "User Service API",
		Version:   "1.0.0",
		Filename:  "user_service.yaml",
	}

	err := openapi.GenerateOpenAPI(&opts, spec)
	if err != nil {
		t.Fatalf("GenerateOpenAPI() error = %v", err)
	}

	yamlFile := filepath.Join(outputPath, "user_service.yaml")
	// #nosec G304 - yamlFile is constructed from t.TempDir() and fixed name in test
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)

	want := `openapi: 3.0.0
info:
  title: User Service API
  version: 1.0.0

paths:
  /users:
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
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
        - name: includeEmail
          in: query
          required: false
          schema:
            type: boolean
        - name: X-Request-ID
          in: header
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GetUserResponse'
        '404':
          description: Not Found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GetUserNotFoundResponse'

components:
  schemas:
    CreateUserRequest:
      type: object
      properties:
        username:
          type: string
        email:
          type: string
        tags:
          type: array
          items:
            type: string
      required:
        - username
        - email
        - tags
    CreateUserResponse:
      type: object
      properties:
        userID:
          type: string
        success:
          type: boolean
      required:
        - userID
        - success
    GetUserNotFoundResponse:
      type: object
      properties:
        message:
          type: string
      required:
        - message
    GetUserResponse:
      type: object
      properties:
        username:
          type: string
        email:
          type: string
        found:
          type: boolean
      required:
        - username
        - email
        - found
`

	if got != want {
		t.Errorf("Generated OpenAPI content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
