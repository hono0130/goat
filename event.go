package goat

import (
	"fmt"
	"reflect"
	"strings"
)

type AbstractEvent interface {
	isEvent() bool
	setRoutingInfo(AbstractStateMachine, AbstractStateMachine)
}

// Event is the base struct that should be embedded in all event implementations.
// It provides the required methods to satisfy the AbstractEvent interface
// and ensures events are properly copyable for the state machine system.
//
// Example:
//
// type MyCustomEvent struct {
// goat.Event[*ProducerStateMachine, *ConsumerStateMachine]
// Payload string
// }
type Event[Sender AbstractStateMachine, Recipient AbstractStateMachine] struct {
	// this is needed to make Event copyable
	_ rune

	sender    Sender
	recipient Recipient
}

func (*Event[Sender, Recipient]) isEvent() bool { return true }

// Sender returns the sender using the concrete type specified by the event's
// type parameters. If the actual sender does not match the requested type, the
// zero value for Sender is returned.
func (e *Event[Sender, Recipient]) Sender() Sender {
	if e == nil {
		var zero Sender
		return zero
	}
	return e.sender
}

// Recipient returns the recipient using the concrete type specified by the
// event's type parameters. If the actual recipient does not match the requested
// type, the zero value for Recipient is returned.
func (e *Event[Sender, Recipient]) Recipient() Recipient {
	if e == nil {
		var zero Recipient
		return zero
	}
	return e.recipient
}

func (e *Event[Sender, Recipient]) setRoutingInfo(sender, recipient AbstractStateMachine) {
	if e == nil {
		return
	}

	if typed, ok := sender.(Sender); ok {
		e.sender = typed
	} else {
		var zero Sender
		e.sender = zero
	}

	if typed, ok := recipient.(Recipient); ok {
		e.recipient = typed
	} else {
		var zero Recipient
		e.recipient = zero
	}
}

type entryEvent struct {
	Event[AbstractStateMachine, AbstractStateMachine]
}

type exitEvent struct {
	Event[AbstractStateMachine, AbstractStateMachine]
}

type transitionEvent struct {
	Event[AbstractStateMachine, AbstractStateMachine]
	To AbstractState
}

type haltEvent struct {
	Event[AbstractStateMachine, AbstractStateMachine]
}

// WARNING: cloneEvent performs shallow copy, so nested pointers are shared
// This is a potential bug - modifications to nested structs will affect both instances
func cloneEvent(event AbstractEvent) AbstractEvent {
	v := reflect.ValueOf(event)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newEvent := reflect.New(v.Type()).Elem()
	newEvent.Set(v)

	return newEvent.Addr().Interface().(AbstractEvent)
}

func sameEvent(e1, e2 AbstractEvent) bool {
	return getEventName(e1) == getEventName(e2)
}

func getEventName(e AbstractEvent) string {
	v := reflect.ValueOf(e)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID EVENT: %v", e))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v.Type().Name()
}

func getEventDetails(e AbstractEvent) string {
	v := reflect.ValueOf(e)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID EVENT: %v", e))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		// WARNING: Pointer fields are intentionally skipped
		// This prevents potential circular references and nil pointer dereferences,
		// but means that important data in pointer fields will not be displayed
		if field.Kind() == reflect.Ptr {
			continue
		}

		if fieldName != "Event" {
			if field.CanInterface() {
				fieldType := field.Type().String()
				fieldValue := field.Interface()
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:%v}", fieldName, fieldType, fieldValue))
			} else {
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:[UNACCESSIBLE]}", fieldName, field.Type().String()))
			}
		}
	}

	if len(fieldDetails) == 0 {
		return noFieldsMessage
	}

	return strings.Join(fieldDetails, ",")
}
