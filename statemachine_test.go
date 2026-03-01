package goat

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testStateWithMap struct {
	State
	Name  string
	Items map[string]int
}

func (s *testStateWithMap) isState() bool { return true }

type testStateWithSlice struct {
	State
	Name   string
	Values []int
}

func (s *testStateWithSlice) isState() bool { return true }

type testStateMachineWithMap struct {
	StateMachine
	Counts map[string]int
}

func (sm *testStateMachineWithMap) isStateMachine() bool { return true }

type testStateMachineWithSlice struct {
	StateMachine
	Items []string
}

func (sm *testStateMachineWithSlice) isStateMachine() bool { return true }

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

func TestNewStateMachineSpec(t *testing.T) {
	t.Run("create spec with test state machine", func(t *testing.T) {
		spec := NewStateMachineSpec(&testStateMachine{})

		if spec.prototype == nil {
			t.Error("Prototype should not be nil")
		}
		if spec.handlerBuilders == nil {
			t.Error("HandlerBuilders should be initialized")
		}
		if len(spec.handlerBuilders) != 0 {
			t.Error("HandlerBuilders should be empty initially")
		}
	})
}

func TestStateMachineSpec_DefineStates(t *testing.T) {
	t.Run("defines states and sets up default handlers", func(t *testing.T) {
		spec := NewStateMachineSpec(&testStateMachine{})

		state1 := newTestState("state1")
		state2 := newTestState("state2")

		result := spec.DefineStates(state1, state2)

		if result != spec {
			t.Error("DefineStates should return self for method chaining")
		}

		if !cmp.Equal(spec.states, []AbstractState{state1, state2}) {
			t.Errorf("States mismatch:\n%s", cmp.Diff([]AbstractState{state1, state2}, spec.states))
		}
		for _, state := range []AbstractState{state1, state2} {
			builders := spec.handlerBuilders[state]
			if len(builders) != 2 {
				t.Errorf("Expected exactly 2 default handlers for state, got %d", len(builders))
			}
		}
	})
}

func TestStateMachineSpec_SetInitialState(t *testing.T) {
	t.Run("sets initial state and returns self", func(t *testing.T) {
		spec := NewStateMachineSpec(&testStateMachine{})
		initialState := newTestState("initial")
		nextState := newTestState("next")
		spec.DefineStates(initialState, nextState)

		result := spec.SetInitialState(initialState)

		if result != spec {
			t.Error("SetInitialState should return self for method chaining")
		}

		if spec.initialState != initialState {
			t.Error("Initial state should be stored")
		}
	})
}

func TestStateMachineSpec_NewInstance(t *testing.T) {
	t.Run("creates new instance with proper initialization", func(t *testing.T) {
		spec := NewStateMachineSpec(&testStateMachine{})
		initialState := newTestState("initial")
		spec.DefineStates(initialState)
		spec.SetInitialState(initialState)

		instance, err := spec.NewInstance()
		if err != nil {
			t.Errorf("NewInstance() returned error: %v", err)
			return
		}

		if instance == nil {
			t.Error("Instance should not be nil")
			return
		}

		if !cmp.Equal(instance.currentState(), initialState) {
			t.Errorf("Initial state mismatch:\n%s", cmp.Diff(initialState, instance.currentState()))
		}

		innerSM := getInnerStateMachine(instance)
		if innerSM.EventHandlers != nil {
			t.Error("Event handlers should be nil initially (built in initialWorld)")
		}

		if innerSM.HandlerBuilders == nil {
			t.Error("Handler builders should be initialized")
		}

		if innerSM.halted {
			t.Error("Instance should not be halted initially")
		}
	})
}

func TestGetInnerStateMachine(t *testing.T) {
	t.Run("valid state machine", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("test"))

		inner := getInnerStateMachine(sm)
		if inner == nil {
			t.Error("getInnerStateMachine() should return non-nil for valid state machine")
		}
	})
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
			StateMachine: StateMachine{
				smID:  "test",
				State: newTestState("initial"),
			},
			Counts: map[string]int{"x": 1, "y": 2},
		}

		cloned := cloneStateMachine(original).(*testStateMachineWithMap)

		if cloned == original {
			t.Error("cloned state machine should be a different instance")
		}

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
			StateMachine: StateMachine{
				smID:  "test",
				State: newTestState("initial"),
			},
			Items: []string{"a", "b"},
		}

		cloned := cloneStateMachine(original).(*testStateMachineWithSlice)

		original.Items[0] = "modified"
		if cloned.Items[0] != "a" {
			t.Error("modifying original slice affected cloned state machine")
		}
	})
}

func TestSameState(t *testing.T) {
	tests := []struct {
		name string
		s1   AbstractState
		s2   AbstractState
		want bool
	}{
		{
			name: "same state instances",
			s1:   newTestState("test"),
			s2:   newTestState("test"),
			want: true,
		},
		{
			name: "different state names",
			s1:   newTestState("test1"),
			s2:   newTestState("test2"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameState(tt.s1, tt.s2)

			if got != tt.want {
				t.Errorf("sameState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStateMachineName(t *testing.T) {
	t.Run("returns correct state machine name", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("test"))
		name := getStateMachineName(sm)

		if name != "testStateMachine" {
			t.Errorf("getStateMachineName() = %v, want %v", name, "testStateMachine")
		}
	})
}

func TestGetStateMachineDetails(t *testing.T) {
	t.Run("returns non-empty details string", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("test"))
		details := getStateMachineDetails(sm)

		if details == "" {
			t.Error("getStateMachineDetails() should return non-empty string")
		}

		if details != noFieldsMessage {
			t.Errorf("getStateMachineDetails() = %v, want %v", details, noFieldsMessage)
		}
	})
}

func TestGetStateDetails(t *testing.T) {
	t.Run("returns state details with name", func(t *testing.T) {
		state := newTestState("test")
		details := getStateDetails(state)

		if details != "{Name:Name,Type:string,Value:test}" {
			t.Errorf("getStateDetails() = %v, want %v", details, "{Name:Name,Type:string,Value:test}")
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
			Name:  "test",
			Items: map[string]int{"a": 1, "b": 2},
		}

		cloned := cloneState(original).(*testStateWithMap)

		if cloned == original {
			t.Error("cloned state should be a different instance")
		}
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
			Name:   "test",
			Values: []int{10, 20, 30},
		}

		cloned := cloneState(original).(*testStateWithSlice)

		original.Values[0] = 99
		if cloned.Values[0] != 10 {
			t.Error("modifying original slice affected cloned state")
		}
	})

	t.Run("nil map and slice remain nil", func(t *testing.T) {
		original := &testStateWithMap{Name: "test"}
		cloned := cloneState(original).(*testStateWithMap)
		if cloned.Items != nil {
			t.Error("nil map should remain nil after clone")
		}
	})
}

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

		if v.Pointer() == copied.Pointer() {
			t.Error("copied map should have different backing pointer")
		}

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

		if v.Pointer() == copied.Pointer() {
			t.Error("copied map should have different backing pointer")
		}

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

		if v.Pointer() == copied.Pointer() {
			t.Error("copied slice should have different backing pointer")
		}

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

		if v.Pointer() == copied.Pointer() {
			t.Error("copied slice should have different backing pointer")
		}

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

func TestDeepCopyStructFields(t *testing.T) {
	t.Run("deep copies map field", func(t *testing.T) {
		original := structWithMapAndSlice{
			Name:  "test",
			Items: map[string]int{"a": 1, "b": 2},
		}
		origMapPtr := reflect.ValueOf(original.Items).Pointer()
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithMapAndSlice)
		if reflect.ValueOf(copied.Items).Pointer() == origMapPtr {
			t.Error("copied map field should have different backing pointer")
		}
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
		origSlicePtr := reflect.ValueOf(original.Values).Pointer()
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithMapAndSlice)
		if reflect.ValueOf(copied.Values).Pointer() == origSlicePtr {
			t.Error("copied slice field should have different backing pointer")
		}
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
		origMapPtr := reflect.ValueOf(original.Inner.Items).Pointer()
		origSlicePtr := reflect.ValueOf(original.Inner.Values).Pointer()
		v := reflect.New(reflect.TypeOf(original)).Elem()
		v.Set(reflect.ValueOf(original))
		deepCopyStructFields(v)

		copied := v.Interface().(structWithNestedStruct)
		if reflect.ValueOf(copied.Inner.Items).Pointer() == origMapPtr {
			t.Error("nested struct's map should have different backing pointer")
		}
		if reflect.ValueOf(copied.Inner.Values).Pointer() == origSlicePtr {
			t.Error("nested struct's slice should have different backing pointer")
		}
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
