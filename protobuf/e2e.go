package protobuf

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/goatx/goat"
)

// E2ETestRecorder provides a simple interface for recording E2E tests
// during manual test execution or exploratory testing.
type E2ETestRecorder struct {
	recorder      *TraceRecorder
	name          string
	description   string
	eventRegistry map[string]reflect.Type
}

// NewE2ETestRecorder creates a new E2E test recorder.
//
// Example:
//
//	recorder := protobuf.NewE2ETestRecorder("user_creation_test", "Test user creation and retrieval")
//	recorder.RegisterEventType(&CreateUserRequest{})
//	recorder.RegisterEventType(&CreateUserResponse{})
func NewE2ETestRecorder(name, description string) *E2ETestRecorder {
	return &E2ETestRecorder{
		recorder:      NewTraceRecorder(),
		name:          name,
		description:   description,
		eventRegistry: make(map[string]reflect.Type),
	}
}

// RegisterEventType registers an event type for later replay.
// This should be called for all input and output message types.
func (e *E2ETestRecorder) RegisterEventType(event AbstractProtobufMessage) {
	typeName := getTypeName(event)
	eventType := reflect.TypeOf(event)
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	e.eventRegistry[typeName] = eventType
}

// GetEventRegistry returns the registered event types.
func (e *E2ETestRecorder) GetEventRegistry() map[string]reflect.Type {
	return e.eventRegistry
}

// Record manually records an RPC call.
// This can be used when automatic tracing is not available.
//
// Example:
//
//	recorder.Record("CreateUser", userService, userService,
//	    &CreateUserRequest{Username: "alice"},
//	    &CreateUserResponse{UserID: "123", Success: true},
//	    0)
func (e *E2ETestRecorder) Record(
	methodName string,
	sender, recipient goat.AbstractStateMachine,
	input, output AbstractProtobufMessage,
	worldID uint64,
) error {
	return e.recorder.RecordRPC(methodName, sender, recipient, input, output, worldID)
}

// GetContext returns a context configured for automatic tracing.
// Use this context when calling RPC handlers to automatically record traces.
//
// Example:
//
//	ctx := recorder.GetContext(context.Background(), worldID)
//	// Now RPC calls with this context will be automatically traced
func (e *E2ETestRecorder) GetContext(ctx context.Context, worldID uint64) context.Context {
	ctx = WithTraceRecorder(ctx, e.recorder)
	ctx = WithWorldID(ctx, worldID)
	return ctx
}

// SaveToFile saves the recorded test case to a JSON file.
//
// Example:
//
//	err := recorder.SaveToFile("testdata/user_creation_test.json")
func (e *E2ETestRecorder) SaveToFile(filepath string) error {
	testCase := e.recorder.ToTestCase(e.name, e.description)
	data, err := SaveTestCase(testCase)
	if err != nil {
		return fmt.Errorf("failed to save test case: %w", err)
	}

	// #nosec G304 - filepath is provided by the caller who controls the destination
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetTestCase returns the current test case.
func (e *E2ETestRecorder) GetTestCase() *E2ETestCase {
	return e.recorder.ToTestCase(e.name, e.description)
}

// GenerateGoTest generates Go test code from the recorded traces.
//
// Example:
//
//	code, err := recorder.GenerateGoTest("main")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("user_service_test.go", []byte(code), 0644)
func (e *E2ETestRecorder) GenerateGoTest(packageName string) (string, error) {
	testCase := e.GetTestCase()
	generator := NewGoTestGenerator(packageName)
	return generator.Generate(testCase)
}

// GenerateGoTestToFile generates Go test code and saves it to a file.
//
// Example:
//
//	err := recorder.GenerateGoTestToFile("main", "user_service_test.go")
func (e *E2ETestRecorder) GenerateGoTestToFile(packageName, filepath string) error {
	testCase := e.GetTestCase()
	generator := NewGoTestGenerator(packageName)
	return generator.GenerateToFile(testCase, filepath)
}

// E2ETestRunner provides a simple interface for replaying E2E tests.
type E2ETestRunner struct {
	eventRegistry map[string]reflect.Type
	stateMachines map[string]goat.AbstractStateMachine
	strictMode    bool
}

// NewE2ETestRunner creates a new E2E test runner.
//
// Example:
//
//	runner := protobuf.NewE2ETestRunner()
//	runner.RegisterEventType(&CreateUserRequest{})
//	runner.RegisterEventType(&CreateUserResponse{})
//	runner.RegisterStateMachine("UserService", userService)
func NewE2ETestRunner() *E2ETestRunner {
	return &E2ETestRunner{
		eventRegistry: make(map[string]reflect.Type),
		stateMachines: make(map[string]goat.AbstractStateMachine),
		strictMode:    true,
	}
}

// RegisterEventType registers an event type for replay.
func (r *E2ETestRunner) RegisterEventType(event AbstractProtobufMessage) {
	typeName := getTypeName(event)
	eventType := reflect.TypeOf(event)
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	r.eventRegistry[typeName] = eventType
}

// RegisterStateMachine registers a state machine instance.
func (r *E2ETestRunner) RegisterStateMachine(id string, sm goat.AbstractStateMachine) {
	r.stateMachines[id] = sm
}

// SetStrictMode enables or disables strict output comparison.
func (r *E2ETestRunner) SetStrictMode(strict bool) {
	r.strictMode = strict
}

// RunFromFile loads and runs a test case from a JSON file.
//
// Example:
//
//	result, err := runner.RunFromFile("testdata/user_creation_test.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Summary())
func (r *E2ETestRunner) RunFromFile(filepath string) (*TestReplayResult, error) {
	// #nosec G304 - filepath is provided by the caller who controls the test file location
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	testCase, err := LoadTestCase(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load test case: %w", err)
	}

	return r.Run(testCase), nil
}

// Run executes a test case and returns the results.
//
// Example:
//
//	result := runner.Run(testCase)
//	if result.FailureCount > 0 {
//	    fmt.Println("Test failed!")
//	    fmt.Println(result.Summary())
//	}
func (r *E2ETestRunner) Run(testCase *E2ETestCase) *TestReplayResult {
	opts := ReplayOptions{
		EventRegistry:      r.eventRegistry,
		StateMachines:      r.stateMachines,
		StrictMode:         r.strictMode,
		StopOnFirstFailure: false,
	}

	return ReplayTest(testCase, opts)
}

// QuickRecord is a convenience function for quickly recording a single RPC interaction.
// This is useful for simple test cases or debugging.
//
// Example:
//
//	testCase := protobuf.QuickRecord(
//	    "simple_test",
//	    "CreateUser",
//	    userService,
//	    &CreateUserRequest{Username: "alice"},
//	    &CreateUserResponse{UserID: "123", Success: true},
//	)
//	data, _ := protobuf.SaveTestCase(testCase)
//	os.WriteFile("test.json", data, 0600)
func QuickRecord(
	testName, methodName string,
	target goat.AbstractStateMachine,
	input, output AbstractProtobufMessage,
) *E2ETestCase {
	recorder := NewTraceRecorder()
	_ = recorder.RecordRPC(methodName, target, target, input, output, 0)
	return recorder.ToTestCase(testName, "")
}
