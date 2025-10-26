package protobuf

import (
	"github.com/goatx/goat"
)

type AbstractProtobufMessage interface {
	isProtobufMessage() bool
	goat.AbstractEvent
}

type ProtobufMessage struct {
	goat.Event[goat.AbstractStateMachine, goat.AbstractStateMachine]
	// this is needed to make ProtobufMessage copyable
	_ rune
}

func (*ProtobufMessage) isProtobufMessage() bool {
	return true
}

type AbstractProtobufServiceSpec interface {
	isProtobufServiceSpec() bool
	GetRPCMethods() []rpcMethod
	GetMessages() map[string]*protoMessage
}

type ProtobufServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	rpcMethods []rpcMethod
	messages   map[string]*protoMessage
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
