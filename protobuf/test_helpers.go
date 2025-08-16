package protobuf

import (
	"github.com/goatx/goat"
)

type TestService1 struct {
	goat.StateMachine
}

type TestService2 struct {
	goat.StateMachine
}

type TestRequest1 struct {
	ProtobufMessage
	Data string
}

type TestResponse1 struct {
	ProtobufMessage
	Result string
}

type TestRequest2 struct {
	ProtobufMessage
	Info string
}

type TestResponse2 struct {
	ProtobufMessage
	Value string
}

type TestRequest3 struct {
	ProtobufMessage
	Input string
}

type TestResponse3 struct {
	ProtobufMessage
	Output string
}

type TestIdleState struct {
	goat.State
}
