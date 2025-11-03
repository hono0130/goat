package protobuf

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/goatx/goat"
)

// RPCTrace represents a single RPC call with its input and output.
// It captures the complete information needed to replay and verify RPC behavior.
type RPCTrace struct {
	// MethodName is the name of the RPC method that was called
	MethodName string `json:"method_name"`

	// Sender is the identifier of the state machine that sent the request
	Sender string `json:"sender"`

	// Recipient is the identifier of the state machine that received the request
	Recipient string `json:"recipient"`

	// InputType is the Go type name of the input message
	InputType string `json:"input_type"`

	// Input is the serialized input message data
	Input map[string]any `json:"input"`

	// OutputType is the Go type name of the output message
	OutputType string `json:"output_type"`

	// Output is the serialized output message data
	Output map[string]any `json:"output"`

	// WorldID is the identifier of the world where this RPC occurred
	WorldID uint64 `json:"world_id"`
}

// E2ETestCase represents a complete e2e test case with multiple RPC traces.
// It can be saved to and loaded from JSON for test replay.
type E2ETestCase struct {
	// Name is a descriptive name for the test case
	Name string `json:"name"`

	// Description provides additional context about what this test validates
	Description string `json:"description,omitempty"`

	// Traces contains all RPC calls in the order they occurred
	Traces []RPCTrace `json:"traces"`
}

// TraceRecorder records RPC calls during model checking execution.
// It can be attached to the model checking process to capture all RPC interactions.
type TraceRecorder struct {
	traces []RPCTrace
}

// NewTraceRecorder creates a new TraceRecorder instance.
func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{
		traces: make([]RPCTrace, 0),
	}
}

// RecordRPC records a single RPC call with its input and output.
// This method should be called by the RPC handler wrapper.
func (tr *TraceRecorder) RecordRPC(
	methodName string,
	sender goat.AbstractStateMachine,
	recipient goat.AbstractStateMachine,
	input AbstractProtobufMessage,
	output AbstractProtobufMessage,
	worldID uint64,
) error {
	inputData, err := serializeMessage(input)
	if err != nil {
		return fmt.Errorf("failed to serialize input: %w", err)
	}

	outputData, err := serializeMessage(output)
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	trace := RPCTrace{
		MethodName: methodName,
		Sender:     getStateMachineID(sender),
		Recipient:  getStateMachineID(recipient),
		InputType:  getTypeName(input),
		Input:      inputData,
		OutputType: getTypeName(output),
		Output:     outputData,
		WorldID:    worldID,
	}

	tr.traces = append(tr.traces, trace)
	return nil
}

// GetTraces returns all recorded traces.
func (tr *TraceRecorder) GetTraces() []RPCTrace {
	return tr.traces
}

// Clear removes all recorded traces.
func (tr *TraceRecorder) Clear() {
	tr.traces = make([]RPCTrace, 0)
}

// ToTestCase converts the recorded traces to an E2ETestCase.
func (tr *TraceRecorder) ToTestCase(name, description string) *E2ETestCase {
	return &E2ETestCase{
		Name:        name,
		Description: description,
		Traces:      tr.traces,
	}
}

// serializeMessage converts a protobuf message to a map for JSON serialization.
func serializeMessage(msg AbstractProtobufMessage) (map[string]any, error) {
	data := make(map[string]any)

	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded (anonymous) fields - these are typically ProtobufMessage or Event
		if field.Anonymous {
			continue
		}

		// Skip fields named "_"
		if field.Name == "_" {
			continue
		}

		// Skip Event and ProtobufMessage fields by type
		if isGoatEventType(field.Type) {
			continue
		}
		if isProtobufMessageType(field.Type) {
			continue
		}

		// Convert the field value to a JSON-compatible type
		data[field.Name] = fieldVal.Interface()
	}

	return data, nil
}

// deserializeMessage populates a protobuf message from a map.
func deserializeMessage(msg AbstractProtobufMessage, data map[string]any) error {
	val := reflect.ValueOf(msg)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("msg must be a pointer")
	}
	val = val.Elem()

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded (anonymous) fields
		if field.Anonymous {
			continue
		}

		// Skip fields named "_"
		if field.Name == "_" {
			continue
		}

		// Skip Event and ProtobufMessage fields by type
		if isGoatEventType(field.Type) {
			continue
		}
		if isProtobufMessageType(field.Type) {
			continue
		}

		// Get the value from the map
		if mapVal, ok := data[field.Name]; ok {
			// Convert the map value to the correct type
			convertedVal := reflect.ValueOf(mapVal)
			if convertedVal.Type().ConvertibleTo(fieldVal.Type()) {
				fieldVal.Set(convertedVal.Convert(fieldVal.Type()))
			} else {
				// Handle slice conversions
				if fieldVal.Kind() == reflect.Slice && convertedVal.Kind() == reflect.Slice {
					slice := reflect.MakeSlice(fieldVal.Type(), convertedVal.Len(), convertedVal.Len())
					for j := 0; j < convertedVal.Len(); j++ {
						elemVal := convertedVal.Index(j)
						if elemVal.Type().ConvertibleTo(fieldVal.Type().Elem()) {
							slice.Index(j).Set(elemVal.Convert(fieldVal.Type().Elem()))
						}
					}
					fieldVal.Set(slice)
				}
			}
		}
	}

	return nil
}

// isProtobufMessageType checks if a type is a ProtobufMessage type.
func isProtobufMessageType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if the type is a struct and has the name "ProtobufMessage"
	// from the protobuf package
	return t.Kind() == reflect.Struct && t.Name() == "ProtobufMessage"
}

// getStateMachineID returns a unique identifier for a state machine.
func getStateMachineID(sm goat.AbstractStateMachine) string {
	if sm == nil {
		return "nil"
	}
	return fmt.Sprintf("%s@%p", getTypeName(sm), sm)
}

// SaveTestCase saves an E2ETestCase to JSON format.
func SaveTestCase(testCase *E2ETestCase) ([]byte, error) {
	return json.MarshalIndent(testCase, "", "  ")
}

// LoadTestCase loads an E2ETestCase from JSON format.
func LoadTestCase(data []byte) (*E2ETestCase, error) {
	var testCase E2ETestCase
	if err := json.Unmarshal(data, &testCase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test case: %w", err)
	}
	return &testCase, nil
}
