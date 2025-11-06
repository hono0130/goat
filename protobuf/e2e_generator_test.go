package protobuf

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat"
)

var update = flag.Bool("update", false, "update golden files")

// Test fixtures for E2E generation
type E2ETestService struct {
	goat.StateMachine
}

type E2ECreateUserRequest struct {
	ProtobufMessage[*E2ETestService, *E2ETestService]
	Username string
	Email    string
}

type E2ECreateUserResponse struct {
	ProtobufMessage[*E2ETestService, *E2ETestService]
	UserID  string
	Success bool
}

type E2EGetUserRequest struct {
	ProtobufMessage[*E2ETestService, *E2ETestService]
	UserID string
}

type E2EGetUserResponse struct {
	ProtobufMessage[*E2ETestService, *E2ETestService]
	Username string
	Email    string
	Found    bool
}

func TestGenerateE2ETest(t *testing.T) {
	tests := []struct {
		name      string
		setupSpec func() *ProtobufServiceSpec[*E2ETestService]
		testCases []TestCase
		golden    string
	}{
		{
			name: "single_test_case",
			setupSpec: func() *ProtobufServiceSpec[*E2ETestService] {
				spec := NewProtobufServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnProtobufMessage(spec, idleState, "CreateUser",
					&E2ECreateUserRequest{}, &E2ECreateUserResponse{},
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) ProtobufResponse[*E2ECreateUserResponse] {
						return ProtobufSendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_123",
							Success: true,
						})
					})

				return spec
			},
			testCases: []TestCase{
				{
					MethodName: "CreateUser",
					Inputs: []AbstractProtobufMessage{
						&E2ECreateUserRequest{
							Username: "alice",
							Email:    "alice@example.com",
						},
					},
				},
			},
			golden: "single_test_case.golden",
		},
		{
			name: "multiple_test_cases",
			setupSpec: func() *ProtobufServiceSpec[*E2ETestService] {
				spec := NewProtobufServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnProtobufMessage(spec, idleState, "CreateUser",
					&E2ECreateUserRequest{}, &E2ECreateUserResponse{},
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) ProtobufResponse[*E2ECreateUserResponse] {
						return ProtobufSendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_456",
							Success: true,
						})
					})

				OnProtobufMessage(spec, idleState, "GetUser",
					&E2EGetUserRequest{}, &E2EGetUserResponse{},
					func(ctx context.Context, req *E2EGetUserRequest, svc *E2ETestService) ProtobufResponse[*E2EGetUserResponse] {
						return ProtobufSendTo(ctx, svc, &E2EGetUserResponse{
							Username: "bob",
							Email:    "bob@example.com",
							Found:    true,
						})
					})

				return spec
			},
			testCases: []TestCase{
				{
					MethodName: "CreateUser",
					Inputs: []AbstractProtobufMessage{
						&E2ECreateUserRequest{
							Username: "bob",
							Email:    "bob@example.com",
						},
					},
				},
				{
					MethodName: "GetUser",
					Inputs: []AbstractProtobufMessage{
						&E2EGetUserRequest{
							UserID: "user_456",
						},
					},
				},
			},
			golden: "multiple_test_cases.golden",
		},
		{
			name: "multiple_inputs_same_method",
			setupSpec: func() *ProtobufServiceSpec[*E2ETestService] {
				spec := NewProtobufServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnProtobufMessage(spec, idleState, "CreateUser",
					&E2ECreateUserRequest{}, &E2ECreateUserResponse{},
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) ProtobufResponse[*E2ECreateUserResponse] {
						return ProtobufSendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_" + req.Username,
							Success: true,
						})
					})

				return spec
			},
			testCases: []TestCase{
				{
					MethodName: "CreateUser",
					Inputs: []AbstractProtobufMessage{
						&E2ECreateUserRequest{
							Username: "alice",
							Email:    "alice@example.com",
						},
						&E2ECreateUserRequest{
							Username: "bob",
							Email:    "bob@example.com",
						},
						&E2ECreateUserRequest{
							Username: "charlie",
							Email:    "charlie@example.com",
						},
					},
				},
			},
			golden: "multiple_inputs_same_method.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			spec := tt.setupSpec()
			tmpDir := t.TempDir()
			outputFile := "generated_test.go"

			// Generate E2E test
			err := GenerateE2ETest(E2ETestOptions{
				Spec:        spec,
				OutputDir:   tmpDir,
				PackageName: "testpkg",
				Filename:    outputFile,
				TestCases:   tt.testCases,
			})
			if err != nil {
				t.Fatalf("GenerateE2ETest() error = %v", err)
			}

			// Read generated file
			generatedPath := filepath.Join(tmpDir, outputFile)
			generated, err := os.ReadFile(generatedPath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			// Compare with golden file
			goldenPath := filepath.Join("testdata", tt.golden)
			if *update {
				// Update golden file
				if err := os.MkdirAll("testdata", 0755); err != nil {
					t.Fatalf("Failed to create testdata directory: %v", err)
				}
				if err := os.WriteFile(goldenPath, generated, 0644); err != nil {
					t.Fatalf("Failed to update golden file: %v", err)
				}
			}

			// Read golden file
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("Failed to read golden file: %v (run with -update to create)", err)
			}

			// Compare
			if string(generated) != string(golden) {
				t.Errorf("Generated code does not match golden file.\nRun 'go test -update' to update golden files.\n\nGenerated:\n%s\n\nGolden:\n%s", string(generated), string(golden))
			}
		})
	}
}
