package goat

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

type AbstractState interface {
	isState() bool
}

type State struct {
	// this is needed to make State copyable
	r rune
}

func (s *State) isState() bool {
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
	setEventHandler(event AbstractEvent, state AbstractState, handler Handler)
	currentState() AbstractState
	setCurrentState(state AbstractState)
	id() string
	SetInitialState(state AbstractState)
	OnEntry(state AbstractState, fs ...OnEntryFunc)
	OnExit(state AbstractState, fs ...OnExitFunc)
	OnEvent(state AbstractState, event AbstractEvent, fs ...OnEventFunc)
	OnTransition(state AbstractState, fs ...OnTransitionFunc)
	OnHalt(state AbstractState, fs ...OnHaltFunc)
	setDefaultHandlers(state AbstractState)
}

type StateMachine struct {
	// immutable fields
	smID          string
	EventHandlers map[AbstractState][]handlerInfo
	// mutable fields
	halted bool
	State  AbstractState
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

func (sm *StateMachine) New(states ...AbstractState) {
	sm.smID = uuid.New().String()
	sm.EventHandlers = make(map[AbstractState][]handlerInfo)
	for _, state := range states {
		sm.setDefaultHandlers(state)
	}
}

func (sm *StateMachine) validate() error {
	if sm.smID == "" {
		return fmt.Errorf("StateMachine doesn't have an id. Please call New() before anything else: %w", ErrInitializeStateMachine)
	}
	if sm.currentState() == nil {
		return fmt.Errorf("StateMachine doesn't have a current state. Please set the current state: %w", ErrInitializeStateMachine)
	}
	return nil
}

func (sm *StateMachine) isStateMachine() bool {
	return true
}

func (sm *StateMachine) setEventHandler(event AbstractEvent, state AbstractState, handler Handler) {
	sm.EventHandlers[state] = append(sm.EventHandlers[state], handlerInfo{
		event:   event,
		handler: handler,
	})
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

func (sm *StateMachine) Goto(state AbstractState, env *Environment) {
	env.enqueueEvent(sm, &ExitEvent{})
	env.enqueueEvent(sm, &TransitionEvent{To: state})
	env.enqueueEvent(sm, &EntryEvent{})
}

func (sm *StateMachine) SendUnary(to AbstractStateMachine, event AbstractEvent, env *Environment) {
	env.enqueueEvent(to, event)
}

func (sm *StateMachine) Halt(to AbstractStateMachine, env *Environment) {
	env.enqueueEvent(to, &HaltEvent{})
}

func (sm *StateMachine) setDefaultHandlers(state AbstractState) {
	defaults := map[AbstractEvent]Handler{
		&TransitionEvent{}: &defaultOnTransitionHandler{},
		&HaltEvent{}:      &defaultOnHaltHandler{},
	}
	for event, h := range defaults {
		sm.setEventHandler(event, state, h)
	}
}

func (sm *StateMachine) OnEntry(state AbstractState, fs ...OnEntryFunc) {
	sm.setEventHandler(&EntryEvent{}, state, &onEntryHandler{fs: fs})
}

func (sm *StateMachine) OnExit(state AbstractState, fs ...OnExitFunc) {
	sm.setEventHandler(&ExitEvent{}, state, &onExitHandler{fs: fs})
}

func (sm *StateMachine) OnEvent(state AbstractState, event AbstractEvent, fs ...OnEventFunc) {
	sm.setEventHandler(event, state, &onEventHandler{fs: fs, event: event})
}

func (sm *StateMachine) OnTransition(state AbstractState, fs ...OnTransitionFunc) {
	sm.setEventHandler(&TransitionEvent{}, state, &onTransitionHandler{fs: fs})
}

func (sm *StateMachine) OnHalt(state AbstractState, fs ...OnHaltFunc) {
	sm.setEventHandler(&HaltEvent{}, state, &onHaltHandler{fs: fs})
}

// getInnerStateMachine extracts the inner state machine from the arbitrary state machine
// that implements AbstractStateMachine.
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

// getStateMachineName returns the type name of the state machine
// that implements AbstractStateMachine.
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

// getStateMachineDetails returns the details of the state machine
// that implements AbstractStateMachine.
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
		return fmt.Sprintf("no fields")
	}

	return strings.Join(fieldDetails, ",")
}

// getStateName returns the type name of the state
// that implements AbstractState.
func getStateName(s AbstractState) string {
	v := reflect.ValueOf(s)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE: %v", s))
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v.Type().Name()
}

// getStateDetails returns the details of the state
// that implements AbstractState.
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
		return fmt.Sprintf("no fields")
	}

	return strings.Join(fieldDetails, ",")
}
