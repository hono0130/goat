package protobuf

import (
	"github.com/goatx/goat"
)

type AbstractMessage interface {
	isMessage() bool
	goat.AbstractEvent
}

type Message[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	_ rune
}

func (*Message[Sender, Recipient]) isMessage() bool {
	return true
}

type AbstractServiceSpec interface {
	isServiceSpec() bool
	GetRPCMethods() []rpcMethod
	GetMessages() map[string]*message
	GetHandlers() map[string]any
	NewStateMachineInstance() (goat.AbstractStateMachine, error)
	GetServiceName() string
}

type ServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	rpcMethods []rpcMethod
	messages   map[string]*message
	handlers   map[string]any
}

func (*ServiceSpec[T]) isServiceSpec() bool {
	return true
}

func (ps *ServiceSpec[T]) GetRPCMethods() []rpcMethod {
	return ps.rpcMethods
}

func (ps *ServiceSpec[T]) GetMessages() map[string]*message {
	return ps.messages
}

func (ps *ServiceSpec[T]) GetHandlers() map[string]any {
	return ps.handlers
}

func (ps *ServiceSpec[T]) NewStateMachineInstance() (goat.AbstractStateMachine, error) {
	return ps.StateMachineSpec.NewInstance()
}

func (ps *ServiceSpec[T]) GetServiceName() string {
	return getServiceTypeName(ps.StateMachineSpec)
}

func (ps *ServiceSpec[T]) addRPCMethod(metadata rpcMethod) {
	ps.rpcMethods = append(ps.rpcMethods, metadata)
}

func (ps *ServiceSpec[T]) addMessage(msg *message) {
	if ps.messages == nil {
		ps.messages = make(map[string]*message)
	}
	ps.messages[msg.Name] = msg
}

type rpcMethod struct {
	ServiceType string
	MethodName  string
	InputType   string
	OutputType  string
}

type message struct {
	Name   string
	Fields []field
}

type field struct {
	Name       string
	Type       string
	Number     int
	IsRepeated bool
}
