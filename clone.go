package goat

import "reflect"

// deepCopyValue creates a deep copy of map and slice values.
// For other kinds, it returns the value unchanged.
func deepCopyValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newMap := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			newMap.SetMapIndex(deepCopyValue(iter.Key()), deepCopyValue(iter.Value()))
		}
		return newMap
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			newSlice.Index(i).Set(deepCopyValue(v.Index(i)))
		}
		return newSlice
	default:
		return v
	}
}

// deepCopyStructFields deep copies all map and slice fields in a struct value,
// including fields in embedded structs.
func deepCopyStructFields(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Kind() {
		case reflect.Map:
			field.Set(deepCopyValue(field))
		case reflect.Slice:
			field.Set(deepCopyValue(field))
		case reflect.Struct:
			deepCopyStructFields(field)
		}
	}
}
