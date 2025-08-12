package goat

// Invariant represents a condition that must hold true throughout
// the state machine execution. It is used to verify system properties
// during architecture testing.
//
// Implementations must ensure that Evaluate returns true when the
// invariant holds for the given world state, and false otherwise.
type Invariant interface {
	Evaluate(w world) bool
}

type invariantFunc func(w world) bool

func (f invariantFunc) Evaluate(w world) bool {
	return f(w)
}

// BoolInvariant creates a simple invariant from a constant boolean value.
// This is useful for creating invariants that always pass (true) or always
// fail (false), typically used for testing or as placeholder invariants.
//
// Parameters:
//   - b: The boolean value that this invariant will always return
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	alwaysPass := goat.BoolInvariant(true)
//	alwaysFail := goat.BoolInvariant(false)
func BoolInvariant(b bool) Invariant {
	return invariantFunc(func(w world) bool {
		return b
	})
}

// NewInvariant creates an invariant for a specific state machine instance.
// It allows checking properties of that particular state machine during
// model exploration and testing.
//
// Parameters:
//   - sm: The state machine instance to create an invariant for
//   - check: A predicate function that returns true if the invariant holds
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	serverInv := goat.NewInvariant(serverSM, func(sm *ServerStateMachine) bool {
//	    return sm.ConnectionCount <= sm.MaxConnections
//	})
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

// Machines provides type-safe access to state machines during invariant evaluation.
// It is used inside check functions to reference multiple state machines.
//
// Implementations return false when the requested state machine does not exist
// in the current world.
type Machines interface {
	Get(sm AbstractStateMachine) (AbstractStateMachine, bool)
}

type machinesImpl struct {
	world world
}

func (m *machinesImpl) Get(sm AbstractStateMachine) (AbstractStateMachine, bool) {
	id := sm.id()
	machine, exists := m.world.env.machines[id]
	if !exists {
		return nil, false
	}
	return machine, true
}

// GetMachine provides type-safe access to a state machine from Machines.
//
// Parameters:
//   - m: Machines accessor provided to the check function
//   - sm: A sample instance used to identify the target machine by ID
//
// Returns the typed state machine and true on success. Returns the zero value
// and false when the machine does not exist or the type does not match.
//
// Example:
//
//	machine, ok := goat.GetMachine(machines, client)
//	if !ok { return false }
//	state := machine.currentState().(*ClientState)
func GetMachine[T AbstractStateMachine](m Machines, sm T) (T, bool) {
	am, ok := m.Get(sm)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := am.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

// NewMultiInvariant creates an invariant that can reference multiple state machines.
// The provided check function receives a Machines accessor.
//
// Parameters:
//   - checkFunc: Predicate that inspects one or more state machines
//   - sms: State machines referenced by the invariant
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	func NewInvariantClientServer(client *Client, server *Server, check func(*Client, *Server) bool) goat.Invariant {
//	    return goat.NewMultiInvariant(func(machines goat.Machines) bool {
//	        c, ok1 := goat.GetMachine(machines, client)
//	        if !ok1 { return false }
//	        s, ok2 := goat.GetMachine(machines, server)
//	        if !ok2 { return false }
//
//	        return check(c, s)
//	    }, client, server)
//	}
//
//	// Usage
//	inv := NewInvariantClientServer(client, server, func(c *Client, s *Server) bool {
//	    // business logic referencing both machines
//	    return c.Server != nil && s != nil
//	})
func NewMultiInvariant(checkFunc func(Machines) bool, sms ...AbstractStateMachine) Invariant {
	return invariantFunc(func(w world) bool {
		m := &machinesImpl{world: w}
		for _, sm := range sms {
			if _, ok := m.Get(sm); !ok {
				return false
			}
		}
		return checkFunc(m)
	})
}

// NewInvariant2 creates an invariant that references two state machines.
//
// Parameters:
//   - sm1, sm2: The state machines to reference
//   - check: A predicate function that returns true if the invariant holds
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	inv := goat.NewInvariant2(client, server, func(c *Client, s *Server) bool {
//	    return true
//	})
func NewInvariant2[T1, T2 AbstractStateMachine](sm1 T1, sm2 T2, check func(T1, T2) bool) Invariant {
	return NewMultiInvariant(func(ms Machines) bool {
		m1, ok := GetMachine(ms, sm1)
		if !ok {
			return false
		}
		m2, ok := GetMachine(ms, sm2)
		if !ok {
			return false
		}
		return check(m1, m2)
	}, sm1, sm2)
}

// NewInvariant3 creates an invariant that references three state machines.
//
// Parameters:
//   - sm1, sm2, sm3: The state machines to reference
//   - check: A predicate function that returns true if the invariant holds
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	inv := goat.NewInvariant3(client, server, db, func(c *Client, s *Server, d *Database) bool {
//	    return true
//	})
func NewInvariant3[T1, T2, T3 AbstractStateMachine](sm1 T1, sm2 T2, sm3 T3, check func(T1, T2, T3) bool) Invariant {
	return NewMultiInvariant(func(ms Machines) bool {
		m1, ok := GetMachine(ms, sm1)
		if !ok {
			return false
		}
		m2, ok := GetMachine(ms, sm2)
		if !ok {
			return false
		}
		m3, ok := GetMachine(ms, sm3)
		if !ok {
			return false
		}
		return check(m1, m2, m3)
	}, sm1, sm2, sm3)
}
