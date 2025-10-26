package goat

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEnvironment_clone(t *testing.T) {
	sm1 := newTestStateMachine(newTestState("state1"))
	sm2 := newTestStateMachine(newTestState("state2"))

	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	event3 := &testEvent{Value: 3}

	original := newTestEnvironment(sm1, sm2)
	original.queue[sm1.id()] = []AbstractEvent{event1, event2}
	original.queue[sm2.id()] = []AbstractEvent{event3}

	cloned := original.clone()

	opts := cmp.Options{
		cmp.AllowUnexported(
			environment{},
			testStateMachine{},
			StateMachine{},
			testState{},
			State{},
			testEvent{},
			Event[*testStateMachine, *testStateMachine]{},
			Event[AbstractStateMachine, AbstractStateMachine]{},
		),
		cmpopts.IgnoreFields(StateMachine{}, "EventHandlers", "HandlerBuilders"),
	}
	if diff := cmp.Diff(original, cloned, opts); diff != "" {
		t.Errorf("environment mismatch (-original +cloned):\n%s", diff)
	}

	for id, sm := range original.machines {
		clonedSm := cloned.machines[id]
		if reflect.ValueOf(sm).Pointer() == reflect.ValueOf(clonedSm).Pointer() {
			t.Errorf("Expected different pointer addresses for machine %s", id)
		}
	}

	for smID, events := range original.queue {
		clonedEvents := cloned.queue[smID]
		for i, event := range events {
			if reflect.ValueOf(event).Pointer() == reflect.ValueOf(clonedEvents[i]).Pointer() {
				t.Errorf("Expected different pointer addresses for event %d of machine %s", i, smID)
			}
		}
	}

}

func TestEnvironment_enqueueEvent(t *testing.T) {
	sm := newTestStateMachine(newTestState("initial"))
	env := newTestEnvironment(sm)
	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	expectedQueue := map[string][]AbstractEvent{
		sm.id(): {event1, event2},
	}

	env.enqueueEvent(sm, event1)
	env.enqueueEvent(sm, event2)

	if diff := cmp.Diff(expectedQueue, env.queue, cmp.AllowUnexported(testEvent{}, Event[*testStateMachine, *testStateMachine]{}, Event[AbstractStateMachine, AbstractStateMachine]{})); diff != "" {
		t.Errorf("Queue mismatch (-expected +actual):\n%s", diff)
	}
}

func TestEnvironment_dequeueEvent(t *testing.T) {
	tests := []struct {
		name          string
		smID          string
		initialQueue  map[string][]AbstractEvent
		expectedEvent *testEvent
		expectedOK    bool
		expectedQueue map[string][]AbstractEvent
	}{
		{
			name: "empty queue",
			smID: "test-sm-id",
			initialQueue: map[string][]AbstractEvent{
				"test-sm-id": {},
			},
			expectedOK: false,
		},
		{
			name:          "non-existent state machine ID",
			smID:          "non-existent-id",
			initialQueue:  make(map[string][]AbstractEvent),
			expectedEvent: nil,
			expectedOK:    false,
		},
		{
			name: "dequeue from queue with events",
			smID: "test-sm-id",
			initialQueue: map[string][]AbstractEvent{
				"test-sm-id": {&testEvent{Value: 1}, &testEvent{Value: 2}, &testEvent{Value: 3}},
			},
			expectedEvent: &testEvent{Value: 1},
			expectedOK:    true,
			expectedQueue: map[string][]AbstractEvent{
				"test-sm-id": {&testEvent{Value: 2}, &testEvent{Value: 3}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := environment{
				queue: tt.initialQueue,
			}

			event, ok := env.dequeueEvent(tt.smID)
			if ok != tt.expectedOK {
				t.Errorf("Expected ok to be %v, but got %v", tt.expectedOK, ok)
			}
			if !tt.expectedOK {
				return
			}

			if diff := cmp.Diff(tt.expectedEvent, event.(*testEvent), cmp.AllowUnexported(testEvent{}, Event[*testStateMachine, *testStateMachine]{}, Event[AbstractStateMachine, AbstractStateMachine]{})); diff != "" {
				t.Errorf("Event mismatch (-expected +actual):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expectedQueue, env.queue, cmp.AllowUnexported(testEvent{}, Event[*testStateMachine, *testStateMachine]{}, Event[AbstractStateMachine, AbstractStateMachine]{})); diff != "" {
				t.Errorf("Queue mismatch (-expected +actual):\n%s", diff)
			}

		})
	}

}

func TestSendTo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (context.Context, *environment, *testStateMachine, *testStateMachine)
		event    AbstractEvent
		wantSize int
		validate func(t *testing.T, queue []AbstractEvent, sender, recipient *testStateMachine)
	}{
		{
			name: "enqueue event for single state machine",
			setup: func(t *testing.T) (context.Context, *environment, *testStateMachine, *testStateMachine) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				ctx := withEnvAndSM(&env, sm)
				return ctx, &env, sm, sm
			},
			event:    &testEvent{Value: 42},
			wantSize: 1,
			validate: func(t *testing.T, queue []AbstractEvent, _ *testStateMachine, _ *testStateMachine) {
				ev, ok := queue[len(queue)-1].(*testEvent)
				if !ok {
					t.Fatalf("queued event mismatch: expected *testEvent, got %T", queue[len(queue)-1])
				}
				if ev.Value != 42 {
					t.Fatalf("queued event value mismatch: got %d, want %d", ev.Value, 42)
				}
			},
		},
		{
			name: "enqueue event for specific machine in multi-environment",
			setup: func(t *testing.T) (context.Context, *environment, *testStateMachine, *testStateMachine) {
				sm1 := newTestStateMachine(newTestState("sm1"))
				sm2 := newTestStateMachine(newTestState("sm2"))
				sm3 := newTestStateMachine(newTestState("sm3"))
				env := newTestEnvironment(sm1, sm2, sm3)
				ctx := withEnvAndSM(&env, sm1)
				return ctx, &env, sm2, sm1
			},
			event:    &testEvent{Value: 123},
			wantSize: 1,
			validate: func(t *testing.T, queue []AbstractEvent, _ *testStateMachine, _ *testStateMachine) {
				ev, ok := queue[len(queue)-1].(*testEvent)
				if !ok {
					t.Fatalf("queued event mismatch: expected *testEvent, got %T", queue[len(queue)-1])
				}
				if ev.Value != 123 {
					t.Fatalf("queued event value mismatch: got %d, want %d", ev.Value, 123)
				}
			},
		},
		{
			name: "append event to existing queue",
			setup: func(t *testing.T) (context.Context, *environment, *testStateMachine, *testStateMachine) {
				sm := newTestStateMachine(newTestState("initial"))
				env := newTestEnvironment(sm)
				env.queue[sm.id()] = []AbstractEvent{&testEvent{Value: 1}}
				ctx := withEnvAndSM(&env, sm)
				return ctx, &env, sm, sm
			},
			event:    &haltEvent{},
			wantSize: 2,
			validate: func(t *testing.T, queue []AbstractEvent, _ *testStateMachine, _ *testStateMachine) {
				if _, ok := queue[len(queue)-1].(*haltEvent); !ok {
					t.Fatalf("queued event mismatch: expected *haltEvent, got %T", queue[len(queue)-1])
				}
			},
		},
		{
			name: "populates routing metadata when sender is available",
			setup: func(t *testing.T) (context.Context, *environment, *testStateMachine, *testStateMachine) {
				sender := newTestStateMachine(newTestState("sender"))
				recipient := newTestStateMachine(newTestState("recipient"))
				env := newTestEnvironment(sender, recipient)
				ctx := withEnvAndSM(&env, sender)
				return ctx, &env, recipient, sender
			},
			event:    &testEvent{Value: 1},
			wantSize: 1,
			validate: func(t *testing.T, queue []AbstractEvent, sender, recipient *testStateMachine) {
				ev := queue[0].(*testEvent)
				if ev.Sender() != sender {
					t.Errorf("expected sender %p, got %p", sender, ev.Sender())
				}
				if ev.Recipient() != recipient {
					t.Errorf("expected recipient %p, got %p", recipient, ev.Recipient())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, env, target, sender := tt.setup(t)
			SendTo(ctx, target, tt.event)

			queue := env.queue[target.id()]
			if len(queue) != tt.wantSize {
				t.Fatalf("expected %d events in queue, got %d", tt.wantSize, len(queue))
			}

			if tt.validate != nil {
				tt.validate(t, queue, sender, target)
				return
			}

			if queue[len(queue)-1] != tt.event {
				t.Fatalf("queued event mismatch: got %v, want %v", queue[len(queue)-1], tt.event)
			}
		})
	}
}

func TestGoto(t *testing.T) {
	tests := []struct {
		name         string
		initialState AbstractState
		targetState  AbstractState
		wantEvents   []AbstractEvent
	}{
		{
			name:         "transition from initial to target state",
			initialState: newTestState("initial"),
			targetState:  newTestState("target"),
			wantEvents: []AbstractEvent{
				&exitEvent{},
				&transitionEvent{To: newTestState("target")},
				&entryEvent{},
			},
		},
		{
			name:         "transition to same state",
			initialState: newTestState("same"),
			targetState:  newTestState("same"),
			wantEvents: []AbstractEvent{
				&exitEvent{},
				&transitionEvent{To: newTestState("same")},
				&entryEvent{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newTestStateMachine(tt.initialState, tt.targetState)
			env := newTestEnvironment(sm)

			ctx := context.WithValue(t.Context(), envKey{}, &env)
			ctx = context.WithValue(ctx, smKey{}, sm)

			Goto(ctx, tt.targetState)

			queue := env.queue[sm.id()]
			opts := cmp.AllowUnexported(
				Event[AbstractStateMachine, AbstractStateMachine]{},
				Event[*testStateMachine, *testStateMachine]{},
				exitEvent{},
				transitionEvent{},
				entryEvent{},
			)
			if !cmp.Equal(queue, tt.wantEvents, opts) {
				t.Errorf("Queue mismatch: %v", cmp.Diff(tt.wantEvents, queue, opts))
			}
		})
	}
}

func TestHalt(t *testing.T) {
	t.Run("enqueues haltEvent to target state machine", func(t *testing.T) {
		sm := newTestStateMachine(newTestState("initial"))
		env := newTestEnvironment(sm)

		ctx := context.WithValue(context.Background(), envKey{}, &env)

		Halt(ctx, sm)

		queue := env.queue[sm.id()]
		wantEvents := []AbstractEvent{&haltEvent{}}

		opts := cmp.AllowUnexported(
			Event[AbstractStateMachine, AbstractStateMachine]{},
			Event[*testStateMachine, *testStateMachine]{},
			haltEvent{},
		)
		if !cmp.Equal(queue, wantEvents, opts) {
			t.Errorf("Queue mismatch: %v", cmp.Diff(wantEvents, queue, opts))
		}
	})
}
