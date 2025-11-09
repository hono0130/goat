package protobuf

import (
	"reflect"
)

// This file contains utility functions for serializing protobuf messages
// and working with Go reflection on state machines.

// serializeMessage converts a protobuf message to a map for code generation.
// Returns map[string]any where 'any' is necessary because protobuf field values
// can be of various types (string, bool, int64, []any, etc.) determined by reflection.
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
