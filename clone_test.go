package goat

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testStateWithMap struct {
	testState
	Items map[string]int
}

type testStateWithSlice struct {
	testState
	Values []int
}

type testStateMachineWithMap struct {
	testStateMachine
	Counts map[string]int
}

type testStateMachineWithSlice struct {
	testStateMachine
	Items []string
}

type testStateWithNestedState struct {
	testState
	Inner testStateWithMap
}

func TestCloneStateMachine(t *testing.T) {
	t.Run("clones state machine with new ID and cloned state", func(t *testing.T) {
		original := newTestStateMachine(newTestState("original"))

		cloned := cloneStateMachine(original)

		if cloned == original {
			t.Error("Cloned state machine should not be the same instance")
		}

		if cloned.id() != original.id() {
			t.Error("Cloned state machine should have same ID (shallow copy)")
		}

		if cloned.currentState() == original.currentState() {
			t.Error("Cloned state should be different instance")
		}

		if !cmp.Equal(cloned.currentState(), original.currentState()) {
			t.Errorf("Cloned state mismatch:\n%s", cmp.Diff(original.currentState(), cloned.currentState()))
		}

		originalInner := getInnerStateMachine(original)
		clonedInner := getInnerStateMachine(cloned)
		if len(originalInner.EventHandlers) != len(clonedInner.EventHandlers) {
			t.Errorf("Handler count mismatch: original=%d, cloned=%d", len(originalInner.EventHandlers), len(clonedInner.EventHandlers))
		}
	})

	t.Run("deep copies map field", func(t *testing.T) {
		original := &testStateMachineWithMap{
			testStateMachine: testStateMachine{StateMachine: StateMachine{
				smID:  "test",
				State: newTestState("initial"),
			}},
			Counts: map[string]int{"x": 1, "y": 2},
		}

		cloned := cloneStateMachine(original).(*testStateMachineWithMap)

		original.Counts["x"] = 99
		original.Counts["z"] = 3
		if cloned.Counts["x"] != 1 {
			t.Error("modifying original map affected cloned state machine")
		}
		if _, exists := cloned.Counts["z"]; exists {
			t.Error("adding to original map affected cloned state machine")
		}
	})

	t.Run("deep copies slice field", func(t *testing.T) {
		original := &testStateMachineWithSlice{
			testStateMachine: testStateMachine{StateMachine: StateMachine{
				smID:  "test",
				State: newTestState("initial"),
			}},
			Items: []string{"a", "b"},
		}

		cloned := cloneStateMachine(original).(*testStateMachineWithSlice)

		original.Items[0] = testModifiedValue
		if cloned.Items[0] != "a" {
			t.Error("modifying original slice affected cloned state machine")
		}
	})
}

func TestCloneState(t *testing.T) {
	t.Run("clones state correctly", func(t *testing.T) {
		original := newTestState("original")
		cloned := cloneState(original)

		if cloned == original {
			t.Error("Cloned state should not be the same instance")
		}

		if !cmp.Equal(cloned, original) {
			t.Errorf("Cloned state mismatch:\n%s", cmp.Diff(original, cloned))
		}
	})

	t.Run("deep copies map field", func(t *testing.T) {
		original := &testStateWithMap{
			testState: testState{Name: "test"},
			Items:     map[string]int{"a": 1, "b": 2},
		}

		cloned := cloneState(original).(*testStateWithMap)

		if cloned.Name != "test" {
			t.Error("Name field should be copied")
		}
		original.Items["a"] = 99
		original.Items["c"] = 3
		if cloned.Items["a"] != 1 {
			t.Error("modifying original map affected cloned state")
		}
		if _, exists := cloned.Items["c"]; exists {
			t.Error("adding to original map affected cloned state")
		}
	})

	t.Run("deep copies slice field", func(t *testing.T) {
		original := &testStateWithSlice{
			testState: testState{Name: "test"},
			Values:    []int{10, 20, 30},
		}

		cloned := cloneState(original).(*testStateWithSlice)

		original.Values[0] = 99
		if cloned.Values[0] != 10 {
			t.Error("modifying original slice affected cloned state")
		}
	})

	t.Run("nil map and slice remain nil", func(t *testing.T) {
		original := &testStateWithMap{testState: testState{Name: "test"}}
		cloned := cloneState(original).(*testStateWithMap)
		if cloned.Items != nil {
			t.Error("nil map should remain nil after clone")
		}
	})
}

func TestDeepCopyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Value
		validate func(*testing.T, reflect.Value, reflect.Value)
	}{
		{
			name:  "nil map returns zero value",
			input: reflect.ValueOf(map[string]int(nil)),
			validate: func(t *testing.T, _, copied reflect.Value) {
				if !copied.IsNil() {
					t.Error("expected nil map to remain nil")
				}
			},
		},
		{
			name:  "copies map with independent backing",
			input: reflect.ValueOf(map[string]int{"a": 1, "b": 2}),
			validate: func(t *testing.T, original, copied reflect.Value) {
				if original.Pointer() == copied.Pointer() {
					t.Error("copied map should have different backing pointer")
				}
			},
		},
		{
			name:  "deep copies nested slices in map values",
			input: reflect.ValueOf(map[string][]int{"nums": {1, 2, 3}}),
			validate: func(t *testing.T, original, copied reflect.Value) {
				if original.Pointer() == copied.Pointer() {
					t.Error("copied map should have different backing pointer")
				}
				origSlice := original.MapIndex(reflect.ValueOf("nums"))
				copiedSlice := copied.MapIndex(reflect.ValueOf("nums"))
				if origSlice.Pointer() == copiedSlice.Pointer() {
					t.Error("nested slice should have different backing pointer")
				}
			},
		},
		{
			name:  "nil slice returns zero value",
			input: reflect.ValueOf([]int(nil)),
			validate: func(t *testing.T, _, copied reflect.Value) {
				if !copied.IsNil() {
					t.Error("expected nil slice to remain nil")
				}
			},
		},
		{
			name:  "copies slice with independent backing",
			input: reflect.ValueOf([]int{1, 2, 3}),
			validate: func(t *testing.T, original, copied reflect.Value) {
				if original.Pointer() == copied.Pointer() {
					t.Error("copied slice should have different backing pointer")
				}
			},
		},
		{
			name:  "deep copies nested maps in slice elements",
			input: reflect.ValueOf([]map[string]int{{"a": 1}, {"b": 2}}),
			validate: func(t *testing.T, original, copied reflect.Value) {
				if original.Pointer() == copied.Pointer() {
					t.Error("copied slice should have different backing pointer")
				}
				if original.Index(0).Pointer() == copied.Index(0).Pointer() {
					t.Error("nested map should have different backing pointer")
				}
			},
		},
		{
			name:  "returns int unchanged",
			input: reflect.ValueOf(42),
			validate: func(t *testing.T, _, copied reflect.Value) {
				if copied.Interface().(int) != 42 {
					t.Error("int value changed")
				}
			},
		},
		{
			name:  "returns string unchanged",
			input: reflect.ValueOf("hello"),
			validate: func(t *testing.T, _, copied reflect.Value) {
				if copied.Interface().(string) != "hello" {
					t.Error("string value changed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := deepCopyValue(tt.input)
			tt.validate(t, tt.input, copied)
		})
	}
}

func TestDeepCopyStructFields(t *testing.T) {
	tests := []struct {
		name     string
		value    func() reflect.Value
		validate func(*testing.T, reflect.Value)
	}{
		func() struct {
			name     string
			value    func() reflect.Value
			validate func(*testing.T, reflect.Value)
		} {
			original := testStateWithMap{
				testState: testState{Name: "test"},
				Items:     map[string]int{"a": 1, "b": 2},
			}
			origMapPtr := reflect.ValueOf(original.Items).Pointer()
			return struct {
				name     string
				value    func() reflect.Value
				validate func(*testing.T, reflect.Value)
			}{
				name: "deep copies map field",
				value: func() reflect.Value {
					v := reflect.New(reflect.TypeOf(original)).Elem()
					v.Set(reflect.ValueOf(original))
					return v
				},
				validate: func(t *testing.T, v reflect.Value) {
					copied := v.Interface().(testStateWithMap)
					if reflect.ValueOf(copied.Items).Pointer() == origMapPtr {
						t.Error("copied map field should have different backing pointer")
					}
				},
			}
		}(),
		func() struct {
			name     string
			value    func() reflect.Value
			validate func(*testing.T, reflect.Value)
		} {
			original := testStateWithSlice{
				testState: testState{Name: "test"},
				Values:    []int{1, 2, 3},
			}
			origSlicePtr := reflect.ValueOf(original.Values).Pointer()
			return struct {
				name     string
				value    func() reflect.Value
				validate func(*testing.T, reflect.Value)
			}{
				name: "deep copies slice field",
				value: func() reflect.Value {
					v := reflect.New(reflect.TypeOf(original)).Elem()
					v.Set(reflect.ValueOf(original))
					return v
				},
				validate: func(t *testing.T, v reflect.Value) {
					copied := v.Interface().(testStateWithSlice)
					if reflect.ValueOf(copied.Values).Pointer() == origSlicePtr {
						t.Error("copied slice field should have different backing pointer")
					}
				},
			}
		}(),
		{
			name: "handles nil map",
			value: func() reflect.Value {
				original := testStateWithMap{testState: testState{Name: "test"}}
				v := reflect.New(reflect.TypeOf(original)).Elem()
				v.Set(reflect.ValueOf(original))
				return v
			},
			validate: func(t *testing.T, v reflect.Value) {
				copied := v.Interface().(testStateWithMap)
				if copied.Items != nil {
					t.Error("nil map should remain nil")
				}
			},
		},
		{
			name: "handles nil slice",
			value: func() reflect.Value {
				original := testStateWithSlice{testState: testState{Name: "test"}}
				v := reflect.New(reflect.TypeOf(original)).Elem()
				v.Set(reflect.ValueOf(original))
				return v
			},
			validate: func(t *testing.T, v reflect.Value) {
				copied := v.Interface().(testStateWithSlice)
				if copied.Values != nil {
					t.Error("nil slice should remain nil")
				}
			},
		},
		func() struct {
			name     string
			value    func() reflect.Value
			validate func(*testing.T, reflect.Value)
		} {
			original := testStateWithNestedState{
				Inner: testStateWithMap{
					Items: map[string]int{"x": 10},
				},
			}
			origMapPtr := reflect.ValueOf(original.Inner.Items).Pointer()
			return struct {
				name     string
				value    func() reflect.Value
				validate func(*testing.T, reflect.Value)
			}{
				name: "recurses into nested structs",
				value: func() reflect.Value {
					v := reflect.New(reflect.TypeOf(original)).Elem()
					v.Set(reflect.ValueOf(original))
					return v
				},
				validate: func(t *testing.T, v reflect.Value) {
					copied := v.Interface().(testStateWithNestedState)
					if reflect.ValueOf(copied.Inner.Items).Pointer() == origMapPtr {
						t.Error("nested struct's map should have different backing pointer")
					}
				},
			}
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.value()
			deepCopyStructFields(v)
			if tt.validate != nil {
				tt.validate(t, v)
			}
		})
	}
}
