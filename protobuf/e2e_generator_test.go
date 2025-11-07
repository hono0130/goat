package protobuf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat"
)

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
	// Setup spec
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

	// Generate E2E tests
	tmpDir := t.TempDir()

	err := GenerateE2ETest(E2ETestOptions{
		OutputDir:   tmpDir,
		PackageName: "testpkg",
		Services: []ServiceTestCase{
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
					{
						MethodName: "GetUser",
						Inputs: []AbstractProtobufMessage{
							&E2EGetUserRequest{UserID: "user_123"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateE2ETest() error = %v", err)
	}

	// List generated files for debugging
	files, _ := os.ReadDir(tmpDir)
	t.Logf("Generated files:")
	for _, f := range files {
		t.Logf("  - %s", f.Name())
	}

	// Verify generated files exist
	mainTestPath := filepath.Join(tmpDir, "main_test.go")
	if _, err := os.Stat(mainTestPath); os.IsNotExist(err) {
		t.Errorf("main_test.go was not generated")
	}

	serviceTestPath := filepath.Join(tmpDir, "e2_etest_service_test.go")
	if _, err := os.Stat(serviceTestPath); os.IsNotExist(err) {
		t.Errorf("e2_etest_service_test.go was not generated")
	}

	// Read and verify main_test.go contains expected content
	mainTestContent, err := os.ReadFile(mainTestPath)
	if err != nil {
		t.Fatalf("Failed to read main_test.go: %v", err)
	}

	mainTestStr := string(mainTestContent)
	if mainTestStr == "" {
		t.Error("main_test.go is empty")
	}

	// Verify it contains TestMain function
	if !contains(mainTestStr, "func TestMain(m *testing.M)") {
		t.Error("main_test.go does not contain TestMain function")
	}

	// Verify it contains client variable
	if !contains(mainTestStr, "var e2_etest_serviceClient") {
		t.Error("main_test.go does not contain client variable")
	}

	// Read and verify service test file
	serviceTestContent, err := os.ReadFile(serviceTestPath)
	if err != nil {
		t.Fatalf("Failed to read service test file: %v", err)
	}

	serviceTestStr := string(serviceTestContent)
	if serviceTestStr == "" {
		t.Error("service test file is empty")
	}

	// Verify it contains test functions
	if !contains(serviceTestStr, "func TestCreateUser(t *testing.T)") {
		t.Error("service test file does not contain TestCreateUser")
	}

	if !contains(serviceTestStr, "func TestGetUser(t *testing.T)") {
		t.Error("service test file does not contain TestGetUser")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
