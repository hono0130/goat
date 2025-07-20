package goat

import (
	"reflect"
	"testing"
)

func TestEnvironmentClone(t *testing.T) {
	sm1 := newTestStateMachine(newTestState("state1"))
	sm2 := newTestStateMachine(newTestState("state2"))

	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	event3 := &testEvent{Value: 3}

	original := newTestEnvironment(sm1, sm2)
	original.queue[sm1.id()] = []AbstractEvent{event1, event2}
	original.queue[sm2.id()] = []AbstractEvent{event3}

	cloned := original.clone()

	assertEnvironmentEqual(t, original, cloned)

	// Verify state machine pointer addresses are different
	for id, sm := range original.machines {
		clonedSm := cloned.machines[id]
		if reflect.ValueOf(sm).Pointer() == reflect.ValueOf(clonedSm).Pointer() {
			t.Errorf("Expected different pointer addresses for machine %s", id)
		}
	}

	// Verify event pointer addresses are different
	for smID, events := range original.queue {
		clonedEvents := cloned.queue[smID]
		for i, event := range events {
			if reflect.ValueOf(event).Pointer() == reflect.ValueOf(clonedEvents[i]).Pointer() {
				t.Errorf("Expected different pointer addresses for event %d of machine %s", i, smID)
			}
		}
	}

}

func TestEnvironmentEnqueueEvent(t *testing.T) {
	sm := newTestStateMachine(newTestState("initial"))
	env := newTestEnvironment(sm)
	event1 := &testEvent{Value: 1}
	event2 := &testEvent{Value: 2}
	expectedQueue := map[string][]AbstractEvent{
		sm.id(): {event1, event2},
	}

	env.enqueueEvent(sm, event1)
	env.enqueueEvent(sm, event2)

	assertQueueEqual(t, expectedQueue, env.queue)
}

func TestEnvironmentDequeueEvent(t *testing.T) {
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
			env := Environment{
				queue: tt.initialQueue,
			}

			event, ok := env.dequeueEvent(tt.smID)
			if ok != tt.expectedOK {
				t.Errorf("Expected ok to be %v, but got %v", tt.expectedOK, ok)
			}
			if !tt.expectedOK {
				return
			}

			assertEventEqual(t, tt.expectedEvent, event.(*testEvent))
			assertQueueEqual(t, tt.expectedQueue, env.queue)

		})
	}

}
