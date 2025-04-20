package goat

import "fmt"

type handlerInfo struct {
	event   AbstractEvent
	handler Handler
}

type Handler interface {
	apply(*StateMachine, AbstractState)
	handle(env Environment, smID string, event AbstractEvent) ([]localState, error)
}

func WithOnEntry(fs ...OnEntryFunc) Handler {
	return &onEntryHandler{fs: fs}
}

func WithOnExit(fs ...OnExitFunc) Handler {
	return &onExitHandler{fs: fs}
}

func WithOnEvent(event AbstractEvent, fs ...OnEventFunc) Handler {
	return &onEventHandler{fs: fs, event: event}
}

func WithOnTransition(fs ...OnTransitionFunc) Handler {
	return &onTransitionHandler{fs: fs}
}

type OnEntryFunc func(this AbstractStateMachine, env *Environment)

type OnExitFunc func(this AbstractStateMachine, env *Environment)

type OnEventFunc func(this AbstractStateMachine, event AbstractEvent, env *Environment)

type OnTransitionFunc func(this AbstractStateMachine, toState AbstractState, env *Environment)

type OnHaltFunc func(this AbstractStateMachine, env *Environment)

type onEntryHandler struct {
	fs []OnEntryFunc
}

func (h *onEntryHandler) apply(sm *StateMachine, state AbstractState) {
	sm.setEventHandler(&EntryEvent{}, state, h)
}

func (h *onEntryHandler) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(sm, &ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onExitHandler struct {
	fs []OnExitFunc
}

func (h *onExitHandler) apply(sm *StateMachine, state AbstractState) {
	sm.setEventHandler(&ExitEvent{}, state, h)
}

func (h *onExitHandler) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(sm, &ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onEventHandler struct {
	fs    []OnEventFunc
	event AbstractEvent
}

func (h *onEventHandler) apply(sm *StateMachine, state AbstractState) {
	sm.setEventHandler(h.event, state, h)
}

func (h *onEventHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(h.event, event) {
		return nil, nil
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(sm, event, &ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onTransitionHandler struct {
	fs []OnTransitionFunc
}

func (h *onTransitionHandler) apply(sm *StateMachine, state AbstractState) {
	sm.setEventHandler(&TransitionEvent{}, state, h)
}

func (h *onTransitionHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&TransitionEvent{}, event) {
		panic(fmt.Sprintf("event is not a TransitionEvent: %v", event))
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(sm, event.(*TransitionEvent).To, &ec)
		sm.setCurrentState(event.(*TransitionEvent).To)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onHaltHandler struct {
	fs []OnHaltFunc
}

func (h *onHaltHandler) apply(sm *StateMachine, state AbstractState) {
	sm.setEventHandler(&HaltEvent{}, state, h)
}

func (h *onHaltHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&HaltEvent{}, event) {
		panic(fmt.Sprintf("event is not a HaltEvent: %v", event))
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(sm, &ec)
		innerSm := getInnerStateMachine(sm)
		innerSm.halted = true
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type defaultOnTransitionHandler struct{}

func (h *defaultOnTransitionHandler) apply(sm *StateMachine, state AbstractState) {
	handlers := sm.EventHandlers[state]
	for _, hi := range handlers {
		if sameEvent(&TransitionEvent{}, hi.event) {
			return
		}
	}
	sm.setEventHandler(&TransitionEvent{}, state, h)
}

func (h *defaultOnTransitionHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&TransitionEvent{}, event) {
		panic(fmt.Sprintf("event is not a TransitionEvent: %v", event))
	}
	ec := env.clone()
	sm := ec.machines[smID]
	sm.setCurrentState(event.(*TransitionEvent).To)
	return []localState{{env: ec}}, nil
}

type defaultOnHaltHandler struct{}

func (h *defaultOnHaltHandler) apply(sm *StateMachine, state AbstractState) {
	handlers := sm.EventHandlers[state]
	for _, hi := range handlers {
		if sameEvent(&HaltEvent{}, hi.event) {
			return
		}
	}
	sm.setEventHandler(&HaltEvent{}, state, h)
}

func (h *defaultOnHaltHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&HaltEvent{}, event) {
		panic(fmt.Sprintf("event is not a HaltEvent: %v", event))
	}
	ec := env.clone()
	sm := ec.machines[smID]
	innerSm := getInnerStateMachine(sm)
	innerSm.halted = true
	return []localState{{env: ec}}, nil
}
