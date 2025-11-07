package protobuf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriter_WriteProtoFile(t *testing.T) {

	tempDir := t.TempDir()
	filename := "empty.proto"
	t.Run("writes empty definitions", func(t *testing.T) {
		w := newFileWriter(tempDir, "test.package", "github.com/test/proto")

		err := w.writeProtoFile(filename, &protoDefinitions{
			Messages: []*protoMessage{},
			Services: []*protoService{},
		})
		if err != nil {
			t.Errorf("WriteProtoFile() error = %v", err)
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

		if !strings.Contains(string(content), "syntax = \"proto3\"") {
			t.Error("generated file missing proto3 syntax declaration")
		}
	})
}

func TestFileWriter_generateFileContent(t *testing.T) {
	tests := []struct {
		name        string
		writer      *fileWriter
		definitions *protoDefinitions
		wantContent string
	}{
		{
			name: "generates minimal content",
			writer: &fileWriter{
				packageName: "test.package",
				goPackage:   "github.com/test/proto",
			},
			definitions: &protoDefinitions{
				Messages: []*protoMessage{},
				Services: []*protoService{},
			},
			wantContent: `syntax = "proto3";

package test.package;

option go_package = "github.com/test/proto";

`,
		},
		{
			name: "generates service with methods",
			writer: &fileWriter{
				packageName: "test.package",
				goPackage:   "github.com/test/proto",
			},
			definitions: &protoDefinitions{
				Messages: []*protoMessage{},
				Services: []*protoService{
					{
						Name: "UserService",
						Methods: []protoMethod{
							{
								Name:       "CreateUser",
								InputType:  "CreateUserRequest",
								OutputType: "CreateUserResponse",
							},
							{
								Name:       "GetUser",
								InputType:  "GetUserRequest",
								OutputType: "GetUserResponse",
							},
						},
					},
				},
			},
			wantContent: `syntax = "proto3";

package test.package;

option go_package = "github.com/test/proto";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
`,
		},
		{
			name: "generates message with fields",
			writer: &fileWriter{
				packageName: "test.package",
				goPackage:   "github.com/test/proto",
			},
			definitions: &protoDefinitions{
				Messages: []*protoMessage{
					{
						Name: "User",
						Fields: []protoField{
							{
								Name:       "UserId",
								Type:       "string",
								Number:     1,
								IsRepeated: false,
							},
							{
								Name:       "UserName",
								Type:       "string",
								Number:     2,
								IsRepeated: false,
							},
							{
								Name:       "Tags",
								Type:       "string",
								Number:     3,
								IsRepeated: true,
							},
						},
					},
				},
				Services: []*protoService{},
			},
			wantContent: `syntax = "proto3";

package test.package;

option go_package = "github.com/test/proto";

message User {
  string user_id = 1;
  string user_name = 2;
  repeated string tags = 3;
}
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

func Test_toSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "converts CamelCase to snake_case",
			input: "CamelCase",
			want:  "camel_case",
		},
		{
			name:  "converts UserId to user_id",
			input: "UserId",
			want:  "user_id",
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
			name:  "converts HTTPRequest",
			input: "HTTPRequest",
			want:  "http_request",
		},
		{
			name:  "converts userID",
			input: "userID",
			want:  "user_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSnakeCase(tt.input)

			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
