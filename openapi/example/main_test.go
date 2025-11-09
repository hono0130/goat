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
	outputPath := filepath.Join(tempDir, "openapi-spec")

	opts := openapi.GenerateOptions{
		OutputDir:   outputPath,
		Title:       "User Service API",
		Version:     "1.0.0",
		Description: "API for managing users",
		Filename:    "user_service.yaml",
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

	// Golden file test
	want := `openapi: 3.0.0
info:
  title: User Service API
  description: API for managing users
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
        '201':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateUserResponse'
  /users/{userId}:
    get:
      operationId: getUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/GetUserRequest'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GetUserResponse'

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
        errorCode:
          type: integer
          format: int64
      required:
        - userID
        - success
        - errorCode
    GetUserRequest:
      type: object
      properties:
        userID:
          type: string
      required:
        - userID
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
