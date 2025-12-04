package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat/protobuf"
)

func TestUserServiceProtobufGeneration(t *testing.T) {
	spec := createUserServiceModel()
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "proto")

	opts := protobuf.GenerateOptions{
		OutputDir:   outputPath,
		PackageName: "user.service",
		GoPackage:   "github.com/goatx/goat/user/proto",
		Filename:    "user_service.proto",
	}

	err := protobuf.Generate(opts, spec)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	protoFile := filepath.Join(outputPath, "user_service.proto")
	// #nosec G304 - protoFile is constructed from t.TempDir() and fixed name in test
	content, err := os.ReadFile(protoFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)

	goldenPath := filepath.Join("proto", "user_service.proto.golden")
	// #nosec G304 - fixed golden path within repo
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile() golden error = %v", err)
	}
	want := string(wantBytes)

	if got != want {
		t.Errorf("Generated proto content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
