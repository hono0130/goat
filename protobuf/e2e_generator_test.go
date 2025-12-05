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

type E2ETestService struct {
	goat.StateMachine
}

type E2ECreateUserRequest struct {
	Message[*E2ETestService, *E2ETestService]
	Username string
	Email    string
}

type E2ECreateUserResponse struct {
	Message[*E2ETestService, *E2ETestService]
	UserID  string
	Success bool
}

type E2EGetUserRequest struct {
	Message[*E2ETestService, *E2ETestService]
	UserID string
}

type E2EGetUserResponse struct {
	Message[*E2ETestService, *E2ETestService]
	Username string
	Email    string
	Found    bool
}

func TestGenerateE2ETest_Golden(t *testing.T) {
	tests := []struct {
		name      string
		setupSpec func() *ServiceSpec[*E2ETestService]
		services  []ServiceTestCase
		goldenDir string
	}{
		{
			name: "single_method_single_input",
			setupSpec: func() *ServiceSpec[*E2ETestService] {
				spec := NewServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnMessage(spec, idleState, "CreateUser",
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) Response[*E2ECreateUserResponse] {
						return SendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_123",
							Success: true,
						})
					})

				return spec
			},
			goldenDir: "single_method_single_input",
		},
		{
			name: "single_method_multiple_inputs",
			setupSpec: func() *ServiceSpec[*E2ETestService] {
				spec := NewServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnMessage(spec, idleState, "CreateUser",
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) Response[*E2ECreateUserResponse] {
						return SendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_" + req.Username,
							Success: true,
						})
					})

				return spec
			},
			goldenDir: "single_method_multiple_inputs",
		},
		{
			name: "multiple_methods",
			setupSpec: func() *ServiceSpec[*E2ETestService] {
				spec := NewServiceSpec(&E2ETestService{})
				idleState := &TestIdleState{}
				spec.DefineStates(idleState).SetInitialState(idleState)

				OnMessage(spec, idleState, "CreateUser",
					func(ctx context.Context, req *E2ECreateUserRequest, svc *E2ETestService) Response[*E2ECreateUserResponse] {
						return SendTo(ctx, svc, &E2ECreateUserResponse{
							UserID:  "user_" + req.Username,
							Success: true,
						})
					})

				OnMessage(spec, idleState, "GetUser",
					func(ctx context.Context, req *E2EGetUserRequest, svc *E2ETestService) Response[*E2EGetUserResponse] {
						return SendTo(ctx, svc, &E2EGetUserResponse{
							Username: "testuser",
							Email:    "test@example.com",
							Found:    true,
						})
					})

				return spec
			},
			goldenDir: "multiple_methods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := tt.setupSpec()

			var services []ServiceTestCase
			if tt.name == "single_method_single_input" {
				services = []ServiceTestCase{
					{
						Spec:           spec,
						ServicePackage: "github.com/example/proto/user",
						Methods: []MethodTestCase{
							{
								MethodName: "CreateUser",
								Inputs: []AbstractMessage{
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
								Inputs: []AbstractMessage{
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
								Inputs: []AbstractMessage{
									&E2ECreateUserRequest{
										Username: "alice",
										Email:    "alice@example.com",
									},
								},
							},
							{
								MethodName: "GetUser",
								Inputs: []AbstractMessage{
									&E2EGetUserRequest{UserID: "user_123"},
								},
							},
						},
					},
				}
			}

			tmpDir := t.TempDir()

			err := GenerateE2ETest(E2ETestOptions{
				OutputDir:   tmpDir,
				PackageName: "testpkg",
				Services:    services,
			})
			if err != nil {
				t.Fatalf("GenerateE2ETest() error = %v", err)
			}

			mainTestPath := filepath.Join(tmpDir, "main_test.go")
			mainTestContent, err := os.ReadFile(mainTestPath)
			if err != nil {
				t.Fatalf("Failed to read main_test.go: %v", err)
			}

			serviceTestPath := filepath.Join(tmpDir, "e2e_test_service_test.go")
			serviceTestContent, err := os.ReadFile(serviceTestPath)
			if err != nil {
				t.Fatalf("Failed to read service test file: %v", err)
			}

			goldenDir := filepath.Join("testdata", tt.goldenDir)
			goldenMainPath := filepath.Join(goldenDir, "main_test.golden")
			goldenServicePath := filepath.Join(goldenDir, "service_test.golden")

			if *update {
				if err := os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatalf("Failed to create golden directory: %v", err)
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

			goldenMain, err := os.ReadFile(goldenMainPath)
			if err != nil {
				t.Fatalf("Failed to read golden main_test file: %v (run with -update to create)", err)
			}

			if string(mainTestContent) != string(goldenMain) {
				t.Errorf("main_test.go does not match golden file.\nRun 'go test -update' to update golden files.")
			}

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
