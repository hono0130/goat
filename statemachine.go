package goat

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

const noFieldsMessage = "no fields"

type handlerBuilder func(smID string) Handler

type handlerBuilderInfo struct {
	event   AbstractEvent
	builder handlerBuilder
}

type StateMachineSpec[T AbstractStateMachine] struct {
	prototype       T
	states          []AbstractState
	initialState    AbstractState
	handlerBuilders map[AbstractState][]handlerBuilderInfo
}

func NewStateMachineSpec[T AbstractStateMachine](prototype T) *StateMachineSpec[T] {
	return &StateMachineSpec[T]{
		prototype:       prototype,
		handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
	}
}

func (spec *StateMachineSpec[T]) DefineStates(states ...AbstractState) *StateMachineSpec[T] {
	spec.states = states
	for _, state := range states {
		spec.setDefaultHandlerBuilders(state)
	}
	return spec
}

func (spec *StateMachineSpec[T]) SetInitialState(state AbstractState) *StateMachineSpec[T] {
	spec.initialState = state
	return spec
}

func (spec *StateMachineSpec[T]) setDefaultHandlerBuilders(state AbstractState) {
	transitionBuilder := func(smID string) Handler {
		return &defaultOnTransitionHandler{}
	}
	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &TransitionEvent{},
		builder: transitionBuilder,
	})

	haltBuilder := func(smID string) Handler {
		return &defaultOnHaltHandler{}
	}
	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &HaltEvent{},
		builder: haltBuilder,
	})
}

func (spec *StateMachineSpec[T]) NewInstance() T {
	instance := cloneStateMachine(spec.prototype).(T)
	innerSM := getInnerStateMachine(instance)

	innerSM.smID = uuid.New().String()
	innerSM.EventHandlers = make(map[AbstractState][]handlerInfo)
	innerSM.State = spec.initialState
	innerSM.halted = false

	for state, builders := range spec.handlerBuilders {
		for _, builderInfo := range builders {
			handler := builderInfo.builder(innerSM.smID)
			innerSM.EventHandlers[state] = append(innerSM.EventHandlers[state], handlerInfo{
				event:   builderInfo.event,
				handler: handler,
			})
		}
	}

	return instance
}

type AbstractState interface {
	isState() bool
}

type State struct {
	// this is needed to make State copyable
	_ rune
}

func (*State) isState() bool {
	return true
}

func cloneState(state AbstractState) AbstractState {
	v := reflect.ValueOf(state)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newState := reflect.New(v.Type()).Elem()
	newState.Set(v)

	return newState.Addr().Interface().(AbstractState)
}

func sameState(s1, s2 AbstractState) bool {
	return getStateDetails(s1) == getStateDetails(s2)
}

type AbstractStateMachine interface {
	isStateMachine() bool
	currentState() AbstractState
	setCurrentState(state AbstractState)
	id() string
	SetInitialState(state AbstractState)
}

type StateMachine struct {
	smID          string
	EventHandlers map[AbstractState][]handlerInfo
	halted        bool
	State         AbstractState
}

func cloneStateMachine(sm AbstractStateMachine) AbstractStateMachine {
	v := reflect.ValueOf(sm)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	smc := reflect.New(v.Type()).Elem()

	smc.Set(v)

	currentStateField := smc.FieldByName("State")
	if currentStateField.IsValid() && !currentStateField.IsZero() {
		state := currentStateField.Interface().(AbstractState)
		currentStateField.Set(reflect.ValueOf(cloneState(state)))
	}

	eventHandlersField := smc.FieldByName("EventHandlers")
	if eventHandlersField.IsValid() && !eventHandlersField.IsZero() {
		oldHandlers := eventHandlersField.Interface().(map[AbstractState][]handlerInfo)
		newHandlers := make(map[AbstractState][]handlerInfo, len(oldHandlers))

		for state, handlers := range oldHandlers {
			newState := cloneState(state)
			// NITS: this is a shallow copy,
			// but it's fine since we don't expect the handlers to be mutated
			newHandlers[newState] = append([]handlerInfo{}, handlers...)
		}

		eventHandlersField.Set(reflect.ValueOf(newHandlers))
	}

	return smc.Addr().Interface().(AbstractStateMachine)
}

func (sm *StateMachine) New(_ ...AbstractState) {
	sm.smID = uuid.New().String()
	sm.EventHandlers = make(map[AbstractState][]handlerInfo)
}

func (*StateMachine) isStateMachine() bool {
	return true
}

func (sm *StateMachine) currentState() AbstractState {
	return sm.State
}

func (sm *StateMachine) setCurrentState(state AbstractState) {
	sm.State = state
}

func (sm *StateMachine) id() string {
	if sm.smID == "" {
		panic("please call New() before anything else")
	}
	return sm.smID
}

func (sm *StateMachine) SetInitialState(state AbstractState) {
	if sm.State != nil {
		panic("initial state already set")
	}
	sm.State = state
}

func getInnerStateMachine(sm AbstractStateMachine) *StateMachine {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("StateMachine")
	if field.IsValid() && field.CanAddr() {
		innerSm, ok := field.Addr().Interface().(*StateMachine)
		if ok {
			return innerSm
		}
	}
	panic("INVALID STATE MACHINE")
}

func getStateMachineName(sm AbstractStateMachine) string {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v.Type().Name()
}

func getStateMachineDetails(sm AbstractStateMachine) string {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if field.Kind() == reflect.Ptr {
			continue
		}

		if fieldName != "StateMachine" {
			if field.CanInterface() {
				fieldType := field.Type().String()
				fieldValue := field.Interface()
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:%v}", fieldName, fieldType, fieldValue))
			} else {
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:[UNACCESSIBLE]}", fieldName, field.Type().String()))
			}
		}
	}

	if len(fieldDetails) == 0 {
		return noFieldsMessage
	}

	return strings.Join(fieldDetails, ",")
}

func getStateDetails(s AbstractState) string {
	v := reflect.ValueOf(s)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE: %v", s))
	}

	t := v.Type()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if fieldName != "State" {
			if field.CanInterface() {
				fieldType := field.Type().String()
				fieldValue := field.Interface()
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:%v}", fieldName, fieldType, fieldValue))
			} else {
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:[UNACCESSIBLE]}", fieldName, field.Type().String()))
			}
		}
	}

	if len(fieldDetails) == 0 {
		return noFieldsMessage
	}

	return strings.Join(fieldDetails, ",")
}
