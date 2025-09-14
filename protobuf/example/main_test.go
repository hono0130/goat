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

	err := protobuf.GenerateProtobuf(opts, spec)
	if err != nil {
		t.Fatalf("GenerateProtobuf() error = %v", err)
	}

	protoFile := filepath.Join(outputPath, "user_service.proto")
	// #nosec G304 - protoFile is constructed from t.TempDir() and fixed name in test
	content, err := os.ReadFile(protoFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)

	// Golden file test
	want := `syntax = "proto3";

package user.service;

option go_package = "github.com/goatx/goat/user/proto";

message CreateUserRequest {
  string username = 1;
  string email = 2;
  repeated string tags = 3;
}

message CreateUserResponse {
  string user_id = 1;
  bool success = 2;
  int64 error_code = 3;
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string username = 1;
  string email = 2;
  bool found = 3;
}

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
`

	if got != want {
		t.Errorf("Generated proto content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
