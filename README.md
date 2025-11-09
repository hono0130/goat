# goat

A Go library for model checking concurrent systems using state machines. goat helps you verify the correctness of distributed systems by exhaustively exploring all possible states and checking conditions/invariants.

## Installation

```bash
go get github.com/goatx/goat
```

Requires Go 1.24.5 or later.

## Quick Example

```go
package main

import (
    "context"
    "log"

    "github.com/goatx/goat"
)

type State struct {
    goat.State
    Name string
}

type MyStateMachine struct {
    goat.StateMachine
    Counter int
}

func main() {
    // Create state machine specification
    spec := goat.NewStateMachineSpec(&MyStateMachine{})
    stateA := &State{Name: "A"}
    stateB := &State{Name: "B"}

    // Define the states that the StateMachine can take and set initial state
    spec.
        DefineStates(stateA, stateB).
        SetInitialState(stateA)

    // Define behavior
    goat.OnEntry(spec, stateA, func(ctx context.Context, sm *MyStateMachine) {
        sm.Counter = 1
        goat.Goto(ctx, stateB)
    })

    goat.OnEntry(spec, stateB, func(ctx context.Context, sm *MyStateMachine) {
        sm.Counter = 2
    })

    // Create instance and run model checking
    sm, err := spec.NewInstance(func(sm *MyStateMachine) {
        sm.Counter = 0
    })
    if err != nil {
        log.Fatal(err)
    }
    cond := goat.NewCondition("counter<=2", sm, func(sm *MyStateMachine) bool {
        return sm.Counter <= 2
    })

    err = goat.Test(
        goat.WithStateMachines(sm),
        goat.WithRules(
            goat.Always(cond),
            goat.WheneverPEventuallyQ(cond, cond),
            goat.EventuallyAlways(cond),
            goat.AlwaysEventually(cond),
        ),
    )
    if err != nil {
        panic(err)
    }
}
```

Pass optional initializer callbacks to `NewInstance` to wire dependencies or
seed default values per instantiation while keeping the spec reusable across
different runs.

## Key Functions

- **`NewStateMachineSpec()`** - Create a state machine specification
- **`OnEntry()`, `OnEvent()`, `OnExit()`** - Register event handlers for each lifecycle events
- **`Goto()`** - Trigger state transitions
- **`SendTo()`** - Send events between state machines
- **`Test()`** - Run model checking with invariant verification
- **`Debug()`** - Output detailed JSON results for debugging
- **`WithStateMachines()`** - Configure which state machines to test
- **`WithRules()`** - Register rules created with helpers like `Always` and `WheneverPEventuallyQ` in one place

## Examples

The `example/` directory contains several complete examples:

- **`simple-transition/`** - Basic state transitions and invariants
- **`client-server/`** - Distributed communication patterns
- **`meeting-room-reservation/`** - Resource contention scenarios
- **`simple-halt/`** - State machine termination
- **`simple-non-deterministic/`** - Non-deterministic behavior modeling

Run any example:

```bash
go run ./example/simple-transition
```

### Multi state machine conditions

- `NewMultiCondition(name string, check func(Machines) bool, sms ...AbstractStateMachine)` — reference multiple machines in one condition
- `NewCondition2` / `NewCondition3` — convenience wrappers for two or three machines
- `Machines` + `GetMachine[T]` — type-safe access to referenced machines during evaluation

```go
cond := goat.NewCondition2("replication", primary, replica, func(p *Storage, r *Storage) bool {
    // Simple consistency: every key in primary exists in replica with the same value
    for key, pv := range p.Data {
        rv, ok := r.Data[key]
        if !ok || rv != pv {
            return false
        }
    }
    return true
})
```
