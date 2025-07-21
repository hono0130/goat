package goat

import (
	"context"
	"fmt"
)

type EventHandler[T AbstractEvent, SM AbstractStateMachine] func(ctx context.Context, event T, sm SM)
type EntryHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)
type ExitHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)
type TransitionHandler[SM AbstractStateMachine] func(ctx context.Context, toState AbstractState, sm SM)
type HaltHandler[SM AbstractStateMachine] func(ctx context.Context, sm SM)

func OnEvent[T AbstractEvent, SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	event T,
	handler EventHandler[T, SM],
) {
	builder := func(smID string) Handler {
		return &eventHandlers{
			fs:    []eventHandler{handleEvent[T, SM](smID, handler)},
			event: event,
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   event,
		builder: builder,
	})
}

func OnEntry[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	handler EntryHandler[SM],
) {
	builder := func(smID string) Handler {
		return &entryHandlers{
			fs: []entryHandler{handleEntry[SM](smID, handler)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &EntryEvent{},
		builder: builder,
	})
}

func OnExit[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	handler ExitHandler[SM],
) {
	builder := func(smID string) Handler {
		return &exitHandlers{
			fs: []exitHandler{handleExit[SM](smID, handler)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &ExitEvent{},
		builder: builder,
	})
}

func OnTransition[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	handler TransitionHandler[SM],
) {
	builder := func(smID string) Handler {
		return &transitionHandlers{
			fs: []transitionHandler{handleTransition[SM](smID, handler)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &TransitionEvent{},
		builder: builder,
	})
}

func OnHalt[SM AbstractStateMachine](
	spec *StateMachineSpec[SM],
	state AbstractState,
	handler HaltHandler[SM],
) {
	builder := func(smID string) Handler {
		return &haltHandlers{
			fs: []haltHandler{handleHalt[SM](smID, handler)},
		}
	}

	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &HaltEvent{},
		builder: builder,
	})
}

func handleEvent[T AbstractEvent, SM AbstractStateMachine](smID string, handler EventHandler[T, SM]) eventHandler {
	return func(event AbstractEvent, env *Environment) {
		typedEvent := event.(T)

		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		handler(ctx, typedEvent, sm)
	}
}

func handleEntry[SM AbstractStateMachine](smID string, handler EntryHandler[SM]) entryHandler {
	return func(env *Environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		handler(ctx, sm)
	}
}

func handleExit[SM AbstractStateMachine](smID string, handler ExitHandler[SM]) exitHandler {
	return func(env *Environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		handler(ctx, sm)
	}
}

func handleTransition[SM AbstractStateMachine](smID string, handler TransitionHandler[SM]) transitionHandler {
	return func(toState AbstractState, env *Environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		handler(ctx, toState, sm)
	}
}

func handleHalt[SM AbstractStateMachine](smID string, handler HaltHandler[SM]) haltHandler {
	return func(env *Environment) {
		machine, exists := env.machines[smID]
		if !exists {
			panic(fmt.Sprintf("StateMachine with ID %s not found in environment", smID))
		}

		sm := machine.(SM)
		ctx := withEnvAndSM(env, sm)

		handler(ctx, sm)
	}
}

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

func (h *entryHandlers) handle(env Environment, _ string, _ AbstractEvent) ([]localState, error) {
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

func (h *exitHandlers) handle(env Environment, _ string, _ AbstractEvent) ([]localState, error) {
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

func (h *eventHandlers) handle(env Environment, _ string, event AbstractEvent) ([]localState, error) {
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
		return nil, nil
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
		return nil, nil
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

//nolint:unparam // error return required by interface
func (*defaultOnTransitionHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&TransitionEvent{}, event) {
		return nil, nil
	}
	ec := env.clone()
	sm := ec.machines[smID]
	sm.setCurrentState(event.(*TransitionEvent).To)
	return []localState{{env: ec}}, nil
}

type defaultOnHaltHandler struct{}

//nolint:unparam // error return required by interface
func (*defaultOnHaltHandler) handle(env Environment, smID string, event AbstractEvent) ([]localState, error) {
	if !sameEvent(&HaltEvent{}, event) {
		return nil, nil
	}
	ec := env.clone()
	sm := ec.machines[smID]
	innerSm := getInnerStateMachine(sm)
	innerSm.halted = true
	return []localState{{env: ec}}, nil
}
