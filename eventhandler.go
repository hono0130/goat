package goat

import "fmt"

type handlerInfo struct {
	event   AbstractEvent
	handler Handler
}

type Handler interface {
	handle(env Environment, smID string, event AbstractEvent) ([]localState, error)
}

type entryHandler func(env *Environment)
type exitHandler func(env *Environment)
type eventHandler func(event AbstractEvent, env *Environment)
type transitionHandler func(toState AbstractState, env *Environment)
type haltHandler func(env *Environment)

type entryHandlers struct {
	fs []entryHandler
}

func (h *entryHandlers) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type exitHandlers struct {
	fs []exitHandler
}

func (h *exitHandlers) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type eventHandlers struct {
	fs    []eventHandler
	event AbstractEvent
}

func (h *eventHandlers) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(h.event, event) {
		return nil, nil
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(event, &ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type transitionHandlers struct {
	fs []transitionHandler
}

func (h *transitionHandlers) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&TransitionEvent{}, event) {
		panic(fmt.Sprintf("event is not a TransitionEvent: %v", event))
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(event.(*TransitionEvent).To, &ec)
		sm.setCurrentState(event.(*TransitionEvent).To)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type haltHandlers struct {
	fs []haltHandler
}

func (h *haltHandlers) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&HaltEvent{}, event) {
		panic(fmt.Sprintf("event is not a HaltEvent: %v", event))
	}
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		sm := ec.machines[smID]
		f(&ec)
		innerSm := getInnerStateMachine(sm)
		innerSm.halted = true
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type defaultOnTransitionHandler struct{}

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
