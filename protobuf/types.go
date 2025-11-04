package protobuf

import (
	"github.com/goatx/goat"
)

type AbstractProtobufMessage interface {
	isProtobufMessage() bool
	goat.AbstractEvent
}

type ProtobufMessage[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	// this is needed to make ProtobufMessage copyable
	_ rune
}

func (*ProtobufMessage[Sender, Recipient]) isProtobufMessage() bool {
	return true
}

type AbstractProtobufServiceSpec interface {
	isProtobufServiceSpec() bool
	GetRPCMethods() []rpcMethod
	GetMessages() map[string]*protoMessage
	GetHandlers() map[string]any
	GetStateMachinePrototype() goat.AbstractStateMachine
	GetSpec() any
}

type ProtobufServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	prototype  T
	rpcMethods []rpcMethod
	messages   map[string]*protoMessage
	handlers   map[string]any // methodName -> handler function
}

func (*ProtobufServiceSpec[T]) isProtobufServiceSpec() bool {
	return true
}

func (ps *ProtobufServiceSpec[T]) GetRPCMethods() []rpcMethod {
	return ps.rpcMethods
}

func (ps *ProtobufServiceSpec[T]) GetMessages() map[string]*protoMessage {
	return ps.messages
}

func (ps *ProtobufServiceSpec[T]) GetHandlers() map[string]any {
	return ps.handlers
}

func (ps *ProtobufServiceSpec[T]) GetStateMachinePrototype() goat.AbstractStateMachine {
	return ps.prototype
}

func (ps *ProtobufServiceSpec[T]) GetSpec() any {
	return ps.StateMachineSpec
}

func (ps *ProtobufServiceSpec[T]) addRPCMethod(metadata rpcMethod) {
	ps.rpcMethods = append(ps.rpcMethods, metadata)
}

func (ps *ProtobufServiceSpec[T]) addMessage(msg *protoMessage) {
	if ps.messages == nil {
		ps.messages = make(map[string]*protoMessage)
	}
	ps.messages[msg.Name] = msg
}

type rpcMethod struct {
	ServiceType string
	MethodName  string
	InputType   string
	OutputType  string
}

type protoMessage struct {
	Name   string
	Fields []protoField
}

type protoField struct {
	Name       string
	Type       string
	Number     int
	IsRepeated bool
}
