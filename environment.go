package goat

import "context"

type Environment struct {
	machines map[string]AbstractStateMachine
	queue    map[string][]AbstractEvent
}

type contextKey string

const envKey contextKey = "environment"
const smKey contextKey = "statemachine"

func withEnvAndSM(env *Environment, sm AbstractStateMachine) context.Context {
	ctx := context.WithValue(context.Background(), envKey, env)
	ctx = context.WithValue(ctx, smKey, sm)
	return ctx
}

func getEnvFromContext(ctx context.Context) *Environment {
	if env, ok := ctx.Value(envKey).(*Environment); ok {
		return env
	}
	panic("Environment not found in context")
}

func getSMFromContext(ctx context.Context) AbstractStateMachine {
	if sm, ok := ctx.Value(smKey).(AbstractStateMachine); ok {
		return sm
	}
	panic("StateMachine not found in context")
}

type localState struct {
	env Environment
}

func (e *Environment) clone() Environment {
	machines := make(map[string]AbstractStateMachine)
	for _, sm := range e.machines {
		smc := cloneStateMachine(sm)
		machines[smc.id()] = smc
	}
	queue := make(map[string][]AbstractEvent)
	for smID, events := range e.queue {
		evsc := make([]AbstractEvent, len(events))
		for i, ev := range events {
			evc := cloneEvent(ev)
			evsc[i] = evc
		}
		queue[smID] = evsc
	}

	ec := Environment{
		machines: machines,
		queue:    queue,
	}
	return ec
}

func (e *Environment) enqueueEvent(target AbstractStateMachine, event AbstractEvent) {
	e.queue[target.id()] = append(e.queue[target.id()], event)
}

func (e *Environment) dequeueEvent(smID string) (AbstractEvent, bool) {
	events, ok := e.queue[smID]
	if !ok {
		return nil, false
	}
	if len(events) == 0 {
		return nil, false
	}

	event := events[0]
	e.queue[smID] = events[1:]
	return event, true
}

func SendTo(ctx context.Context, target AbstractStateMachine, event AbstractEvent) {
	env := getEnvFromContext(ctx)
	env.enqueueEvent(target, event)
}

func Goto(ctx context.Context, state AbstractState) {
	env := getEnvFromContext(ctx)
	sm := getSMFromContext(ctx)
	env.enqueueEvent(sm, &ExitEvent{})
	env.enqueueEvent(sm, &TransitionEvent{To: state})
	env.enqueueEvent(sm, &EntryEvent{})
}

func Halt(ctx context.Context, target AbstractStateMachine) {
	env := getEnvFromContext(ctx)
	env.enqueueEvent(target, &HaltEvent{})
}
