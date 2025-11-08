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
	// GetHandlers returns a map of method names to handler functions.
	// 'any' is necessary because each handler has a different signature based on request/response types.
	GetHandlers() map[string]any
	NewStateMachineInstance() (goat.AbstractStateMachine, error)
	GetServiceName() string
}

type ProtobufServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	rpcMethods []rpcMethod
	messages   map[string]*protoMessage
	// handlers stores handler functions for each RPC method.
	// 'any' is necessary because each handler has a different signature.
	handlers map[string]any // methodName -> handler function
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

func (ps *ProtobufServiceSpec[T]) NewStateMachineInstance() (goat.AbstractStateMachine, error) {
	return ps.StateMachineSpec.NewInstance()
}

func (ps *ProtobufServiceSpec[T]) GetServiceName() string {
	return getServiceTypeName(ps.StateMachineSpec)
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
