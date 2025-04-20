package goat

type Environment struct {
	machines map[string]AbstractStateMachine
	queue    map[string][]AbstractEvent
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
