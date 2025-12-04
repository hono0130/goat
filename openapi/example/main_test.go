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

	err := openapi.Generate(&opts, spec)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	yamlFile := filepath.Join(outputPath, "user_service.yaml")
	// #nosec G304 - yamlFile is constructed from t.TempDir() and fixed name in test
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)

	goldenPath := filepath.Join("openapi", "user_service.yaml.golden")
	// #nosec G304 - fixed golden path within repo
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile() golden error = %v", err)
	}
	want := string(wantBytes)

	if got != want {
		t.Errorf("Generated OpenAPI content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
