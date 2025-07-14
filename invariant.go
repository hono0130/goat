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

// NewInvariant creates a type-safe invariant for a specific state machine type
func NewInvariant[T AbstractStateMachine](sm T, check func(T) bool) Invariant {
	id := sm.id()
	return invariantFunc(func(w world) bool {
		machine, exists := w.env.machines[id]
		if !exists {
			return false
		}
		typedMachine, ok := machine.(T)
		if !ok {
			return false
		}
		return check(typedMachine)
	})
}
