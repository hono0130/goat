package goat

import (
	"reflect"
	"testing"
)

func TestDeepCopyValue(t *testing.T) {
	t.Run("nil map returns zero value", func(t *testing.T) {
		var m map[string]int
		v := reflect.ValueOf(m)
		copied := deepCopyValue(v)
		if !copied.IsNil() {
			t.Error("expected nil map to remain nil")
		}
	})

	t.Run("copies map with independent backing", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		v := reflect.ValueOf(m)
		copied := deepCopyValue(v)

		copiedMap := copied.Interface().(map[string]int)
		if len(copiedMap) != 2 || copiedMap["a"] != 1 || copiedMap["b"] != 2 {
			t.Errorf("copied map content mismatch: %v", copiedMap)
		}

		m["a"] = 99
		m["c"] = 3
		if copiedMap["a"] != 1 {
			t.Error("modifying original map affected the copy")
		}
		if _, exists := copiedMap["c"]; exists {
			t.Error("adding to original map affected the copy")
		}
	})

	t.Run("deep copies nested slices in map values", func(t *testing.T) {
		m := map[string][]int{"nums": {1, 2, 3}}
		v := reflect.ValueOf(m)
		copied := deepCopyValue(v)

		copiedMap := copied.Interface().(map[string][]int)
		m["nums"][0] = 99
		if copiedMap["nums"][0] != 1 {
			t.Error("modifying nested slice in original map affected the copy")
		}
	})

	t.Run("nil slice returns zero value", func(t *testing.T) {
		var s []int
		v := reflect.ValueOf(s)
		copied := deepCopyValue(v)
		if !copied.IsNil() {
			t.Error("expected nil slice to remain nil")
		}
	})

	t.Run("copies slice with independent backing", func(t *testing.T) {
		s := []int{1, 2, 3}
		v := reflect.ValueOf(s)
		copied := deepCopyValue(v)

		copiedSlice := copied.Interface().([]int)
		if len(copiedSlice) != 3 || copiedSlice[0] != 1 {
			t.Errorf("copied slice content mismatch: %v", copiedSlice)
		}

		s[0] = 99
		if copiedSlice[0] != 1 {
			t.Error("modifying original slice affected the copy")
		}
	})

	t.Run("deep copies nested maps in slice elements", func(t *testing.T) {
		s := []map[string]int{{"a": 1}, {"b": 2}}
		v := reflect.ValueOf(s)
		copied := deepCopyValue(v)

		copiedSlice := copied.Interface().([]map[string]int)
		s[0]["a"] = 99
		if copiedSlice[0]["a"] != 1 {
			t.Error("modifying nested map in original slice affected the copy")
		}
	})

	t.Run("returns int unchanged", func(t *testing.T) {
		v := reflect.ValueOf(42)
		copied := deepCopyValue(v)
		if copied.Interface().(int) != 42 {
			t.Error("int value changed")
		}
	})

	t.Run("returns string unchanged", func(t *testing.T) {
		v := reflect.ValueOf("hello")
		copied := deepCopyValue(v)
		if copied.Interface().(string) != "hello" {
			t.Error("string value changed")
		}
	})
}

type structWithMapAndSlice struct {
	State
	Name   string
	Items  map[string]int
	Values []int
}

type structWithNestedStruct struct {
	State
	Inner structWithMapAndSlice
}

func TestDeepCopyStructFields(t *testing.T) {
	t.Run("deep copies map field", func(t *testing.T) {
		original := structWithMapAndSlice{
			Name:  "test",
			Items: map[string]int{"a": 1, "b": 2},
		}
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithMapAndSlice)
		original.Items["a"] = 99
		if copied.Items["a"] != 1 {
			t.Error("modifying original map affected the deep copied struct")
		}
	})

	t.Run("deep copies slice field", func(t *testing.T) {
		original := structWithMapAndSlice{
			Name:   "test",
			Values: []int{1, 2, 3},
		}
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithMapAndSlice)
		original.Values[0] = 99
		if copied.Values[0] != 1 {
			t.Error("modifying original slice affected the deep copied struct")
		}
	})

	t.Run("handles nil map and slice", func(t *testing.T) {
		original := structWithMapAndSlice{Name: "test"}
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithMapAndSlice)
		if copied.Items != nil {
			t.Error("nil map should remain nil")
		}
		if copied.Values != nil {
			t.Error("nil slice should remain nil")
		}
	})

	t.Run("recurses into nested structs", func(t *testing.T) {
		original := structWithNestedStruct{
			Inner: structWithMapAndSlice{
				Items:  map[string]int{"x": 10},
				Values: []int{5, 6},
			},
		}
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithNestedStruct)
		original.Inner.Items["x"] = 99
		original.Inner.Values[0] = 99
		if copied.Inner.Items["x"] != 10 {
			t.Error("modifying nested struct's map affected the copy")
		}
		if copied.Inner.Values[0] != 5 {
			t.Error("modifying nested struct's slice affected the copy")
		}
	})

	t.Run("works with pointer to struct", func(t *testing.T) {
		original := &structWithMapAndSlice{
			Items: map[string]int{"k": 42},
		}
		v := reflect.ValueOf(original)
		deepCopyStructFields(v)

		original.Items["k"] = 0
		if original.Items["k"] != 0 {
			t.Error("unexpected behavior")
		}
	})
}

