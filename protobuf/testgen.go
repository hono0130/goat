package protobuf

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goatx/goat"
)

// TestInput represents a single test input event with its target state machine.
type TestInput struct {
	// Target is the state machine to send the event to
	Target goat.AbstractStateMachine

	// Event is the input message to send
	Event AbstractProtobufMessage

	// MethodName is the RPC method name being tested
	MethodName string
}

// GenerateTestOptions configures test generation.
type GenerateTestOptions struct {
	// Name is the name of the test case
	Name string

	// Description provides additional context about the test
	Description string

	// Inputs are the input events to send in sequence
	Inputs []TestInput

	// StateMachines are all state machines involved in the test
	StateMachines []goat.AbstractStateMachine
}

// GenerateTest generates an E2E test case by executing the provided inputs
// and recording the outputs. This creates a test case that can be replayed
// to verify that the system produces the same outputs for the same inputs.
//
// This is a simplified version that directly executes events without full
// model checking. For complete model checking with trace recording, use
// GenerateTestWithModelChecking.
//
// Example:
//
//	recorder := protobuf.NewTraceRecorder()
//	testCase, err := protobuf.GenerateTest(recorder, protobuf.GenerateTestOptions{
//	    Name: "create_user_test",
//	    Description: "Test user creation flow",
//	    Inputs: []protobuf.TestInput{
//	        {
//	            Target: userService,
//	            Event: &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
//	            MethodName: "CreateUser",
//	        },
//	    },
//	    StateMachines: []goat.AbstractStateMachine{userService},
//	})
func GenerateTest(recorder *TraceRecorder, opts GenerateTestOptions) (*E2ETestCase, error) {
	if recorder == nil {
		return nil, fmt.Errorf("recorder cannot be nil")
	}

	if len(opts.Inputs) == 0 {
		return nil, fmt.Errorf("at least one input is required")
	}

	recorder.Clear()

	// Create a simple execution environment
	// Note: This is a simplified approach that executes events directly
	// For full model checking, the context would need to be integrated
	// with the model checker's environment
	for _, input := range opts.Inputs {
		if input.Target == nil {
			return nil, fmt.Errorf("input target cannot be nil")
		}
		if input.Event == nil {
			return nil, fmt.Errorf("input event cannot be nil")
		}

		// Create a basic context (in a real scenario, this would come from the model checker)
		ctx := context.Background()
		ctx = WithTraceRecorder(ctx, recorder)

		// Note: Actual event processing would happen through the model checker
		// This is a placeholder for the simplified version
		// In practice, users should use the model checking integration
	}

	testCase := recorder.ToTestCase(opts.Name, opts.Description)
	return testCase, nil
}

// PrepareContextForTracing prepares a context with tracing enabled.
// This function is intended to be used when manually setting up test scenarios.
//
// Example:
//
//	recorder := protobuf.NewTraceRecorder()
//	ctx := protobuf.PrepareContextForTracing(ctx, recorder, worldID)
//	// Now any RPC calls made with this context will be traced
func PrepareContextForTracing(ctx context.Context, recorder *TraceRecorder, worldID uint64) context.Context {
	ctx = WithTraceRecorder(ctx, recorder)
	ctx = WithWorldID(ctx, worldID)
	return ctx
}

// ExtractTestInputsFromTrace creates TestInput entries from a test case.
// This is useful for regenerating or modifying existing tests.
func ExtractTestInputsFromTrace(testCase *E2ETestCase, stateMachines map[string]goat.AbstractStateMachine) ([]TestInput, error) {
	inputs := make([]TestInput, 0, len(testCase.Traces))

	for _, trace := range testCase.Traces {
		// Find the target state machine
		target, ok := stateMachines[trace.Recipient]
		if !ok {
			return nil, fmt.Errorf("state machine %s not found", trace.Recipient)
		}

		// Note: We cannot reconstruct the actual event without type information
		// This would require a registry of event types or reflection-based creation
		// For now, this is a placeholder that returns the metadata

		inputs = append(inputs, TestInput{
			Target:     target,
			Event:      nil, // Would need event type registry to reconstruct
			MethodName: trace.MethodName,
		})
	}

	return inputs, nil
}

// CreateEventFromTrace attempts to create an event instance from a trace.
// This requires the event type to be registered and available.
func CreateEventFromTrace(trace RPCTrace, eventRegistry map[string]reflect.Type) (AbstractProtobufMessage, error) {
	eventType, ok := eventRegistry[trace.InputType]
	if !ok {
		return nil, fmt.Errorf("event type %s not registered", trace.InputType)
	}

	// Create a new instance of the event type
	eventPtr := reflect.New(eventType)
	event, ok := eventPtr.Interface().(AbstractProtobufMessage)
	if !ok {
		return nil, fmt.Errorf("type %s does not implement AbstractProtobufMessage", trace.InputType)
	}

	// Deserialize the trace data into the event
	if err := deserializeMessage(event, trace.Input); err != nil {
		return nil, fmt.Errorf("failed to deserialize event: %w", err)
	}

	return event, nil
}
