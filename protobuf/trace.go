package protobuf

import (
	"reflect"
)

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

		if !field.IsExported() || field.Anonymous || field.Name == "_" {
			continue
		}
		if isGoatEventType(field.Type) || isProtobufMessageType(field.Type) {
			continue
		}

		data[field.Name] = fieldVal.Interface()
	}

	return data, nil
}

func isProtobufMessageType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t.Name() == "ProtobufMessage"
}
