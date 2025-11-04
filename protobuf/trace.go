package protobuf

import (
	"fmt"
	"reflect"

	"github.com/goatx/goat"
)

// E2ETestCase represents a single e2e test case.
type E2ETestCase struct {
	// MethodName is the RPC method being tested
	MethodName string

	// InputType is the Go type name of the input message
	InputType string

	// Input is the serialized input message data
	Input map[string]any

	// OutputType is the Go type name of the output message
	OutputType string

	// Output is the serialized output message data
	Output map[string]any
}

// serializeMessage converts a protobuf message to a map for code generation.
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
