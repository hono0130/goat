package goat

import "fmt"

type handlerInfo struct {
	event   AbstractEvent
	handler Handler
}

type Handler interface {
	handle(env Environment, smID string, event AbstractEvent) ([]localState, error)
}

type OnEntryFunc func(env *Environment)

type OnExitFunc func(env *Environment)

type OnEventFunc func(event AbstractEvent, env *Environment)

type OnTransitionFunc func(toState AbstractState, env *Environment)

type OnHaltFunc func(env *Environment)

type onEntryHandler struct {
	fs []OnEntryFunc
}

func (h *onEntryHandler) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onExitHandler struct {
	fs []OnExitFunc
}

func (h *onExitHandler) handle(env Environment, smID string, _ AbstractEvent) ([]localState, error) {
	lss := make([]localState, 0)
	for _, f := range h.fs {
		ec := env.clone()
		f(&ec)
		lss = append(lss, localState{env: ec})
	}
	return lss, nil
}

type onEventHandler struct {
	fs    []OnEventFunc
	event AbstractEvent
}

func (h *onEventHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
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

type onTransitionHandler struct {
	fs []OnTransitionFunc
}

func (h *onTransitionHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
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

type onHaltHandler struct {
	fs []OnHaltFunc
}

func (h *onHaltHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
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
