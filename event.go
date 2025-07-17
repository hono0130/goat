package goat

import (
	"fmt"
	"reflect"
	"strings"
)

type AbstractEvent interface {
	isEvent() bool
}

type Event struct {
	// this is needed to make Event copyable
	_ rune
}

func (*Event) isEvent() bool {
	return true
}

type (
	// EntryEvent is an event that is triggered
	// when a state machine enters a state.
	EntryEvent struct {
		Event
	}

	// ExitEvent is an event that is triggered
	// when a state machine exits a state.
	ExitEvent struct {
		Event
	}

	// TransitionEvent is an event that is triggered
	// when a state machine transitions from one state to another.
	TransitionEvent struct {
		Event
		To AbstractState
	}

	// HaltEvent is an event that is triggered
	// when a state machine halts.
	HaltEvent struct {
		Event
	}
)

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
