package goat

type Invariant interface {
	Evaluate(w world) bool
}

type invariantFunc func(w world) bool

func (f invariantFunc) Evaluate(w world) bool {
	return f(w)
}

func BoolInvariant(b bool) Invariant {
	return invariantFunc(func(w world) bool {
		return b
	})
}

// Ref represents a reference to a state machine by ID
type Ref struct {
	id string
}

// statemachine retrieves the state machine from the world
func (r *Ref) statemachine(w world) (AbstractStateMachine, bool) {
	sm, exists := w.env.machines[r.id]
	if !exists {
		return nil, false
	}
	return sm, true
}

// ToRef creates a reference to a state machine that can be evaluated in invariants
func ToRef(sm AbstractStateMachine) *Ref {
	return &Ref{id: sm.id()}
}

// Invariant creates an invariant that checks a condition on the state machine
func (r *Ref) Invariant(check func(sm AbstractStateMachine) bool) Invariant {
	return invariantFunc(func(w world) bool {
		sm, exists := r.statemachine(w)
		if !exists {
			return false
		}
		return check(sm)
	})
}
