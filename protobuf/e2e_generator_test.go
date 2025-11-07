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

func TestGenerateE2ETest_Golden(t *testing.T) {
	tests := []struct {
		name           string
		setupSpec      func() *ProtobufServiceSpec[*E2ETestService]
		services       []ServiceTestCase
		goldenMainTest string
		goldenService  string
	}{
		{
			name: "single_method_single_input",
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
			goldenMainTest: "single_method_single_input_main_test.golden",
			goldenService:  "single_method_single_input_service_test.golden",
		},
		{
			name: "single_method_multiple_inputs",
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
			goldenMainTest: "single_method_multiple_inputs_main_test.golden",
			goldenService:  "single_method_multiple_inputs_service_test.golden",
		},
		{
			name: "multiple_methods",
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

				OnProtobufMessage(spec, idleState, "GetUser",
					&E2EGetUserRequest{}, &E2EGetUserResponse{},
					func(ctx context.Context, req *E2EGetUserRequest, svc *E2ETestService) ProtobufResponse[*E2EGetUserResponse] {
						return ProtobufSendTo(ctx, svc, &E2EGetUserResponse{
							Username: "testuser",
							Email:    "test@example.com",
							Found:    true,
						})
					})

				return spec
			},
			goldenMainTest: "multiple_methods_main_test.golden",
			goldenService:  "multiple_methods_service_test.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup spec
			spec := tt.setupSpec()

			// Build services configuration
			var services []ServiceTestCase
			if tt.name == "single_method_single_input" {
				services = []ServiceTestCase{
					{
						Spec:           spec,
						ServicePackage: "github.com/example/proto/user",
						Methods: []MethodTestCase{
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
					},
				}
			} else if tt.name == "single_method_multiple_inputs" {
				services = []ServiceTestCase{
					{
						Spec:           spec,
						ServicePackage: "github.com/example/proto/user",
						Methods: []MethodTestCase{
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
								},
							},
						},
					},
				}
			} else if tt.name == "multiple_methods" {
				services = []ServiceTestCase{
					{
						Spec:           spec,
						ServicePackage: "github.com/example/proto/user",
						Methods: []MethodTestCase{
							{
								MethodName: "CreateUser",
								Inputs: []AbstractProtobufMessage{
									&E2ECreateUserRequest{
										Username: "alice",
										Email:    "alice@example.com",
									},
								},
							},
							{
								MethodName: "GetUser",
								Inputs: []AbstractProtobufMessage{
									&E2EGetUserRequest{UserID: "user_123"},
								},
							},
						},
					},
				}
			}

			// Generate E2E tests
			tmpDir := t.TempDir()

			err := GenerateE2ETest(E2ETestOptions{
				OutputDir:   tmpDir,
				PackageName: "testpkg",
				Services:    services,
			})
			if err != nil {
				t.Fatalf("GenerateE2ETest() error = %v", err)
			}

			// Read generated main_test.go
			mainTestPath := filepath.Join(tmpDir, "main_test.go")
			mainTestContent, err := os.ReadFile(mainTestPath)
			if err != nil {
				t.Fatalf("Failed to read main_test.go: %v", err)
			}

			// Read generated service test file
			serviceTestPath := filepath.Join(tmpDir, "e2e_test_service_test.go")
			serviceTestContent, err := os.ReadFile(serviceTestPath)
			if err != nil {
				t.Fatalf("Failed to read service test file: %v", err)
			}

			// Compare with golden files
			goldenMainPath := filepath.Join("testdata", tt.goldenMainTest)
			goldenServicePath := filepath.Join("testdata", tt.goldenService)

			if *update {
				// Update golden files
				if err := os.MkdirAll("testdata", 0755); err != nil {
					t.Fatalf("Failed to create testdata directory: %v", err)
				}
				if err := os.WriteFile(goldenMainPath, mainTestContent, 0644); err != nil {
					t.Fatalf("Failed to update golden main_test file: %v", err)
				}
				if err := os.WriteFile(goldenServicePath, serviceTestContent, 0644); err != nil {
					t.Fatalf("Failed to update golden service test file: %v", err)
				}
				t.Logf("Updated golden files for %s", tt.name)
				return
			}

			// Read and compare main_test.go
			goldenMain, err := os.ReadFile(goldenMainPath)
			if err != nil {
				t.Fatalf("Failed to read golden main_test file: %v (run with -update to create)", err)
			}

			if string(mainTestContent) != string(goldenMain) {
				t.Errorf("main_test.go does not match golden file.\nRun 'go test -update' to update golden files.")
			}

			// Read and compare service test file
			goldenService, err := os.ReadFile(goldenServicePath)
			if err != nil {
				t.Fatalf("Failed to read golden service test file: %v (run with -update to create)", err)
			}

			if string(serviceTestContent) != string(goldenService) {
				t.Errorf("service test file does not match golden file.\nRun 'go test -update' to update golden files.")
			}
		})
	}
}
