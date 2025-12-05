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
	Message[*TestService1, *TestService1]
	Data string
}

type TestResponse1 struct {
	Message[*TestService1, *TestService1]
	Result string
}

type TestRequest2 struct {
	Message[*TestService1, *TestService1]
	Info string
}

type TestResponse2 struct {
	Message[*TestService1, *TestService1]
	Value string
}

type TestIdleState struct {
	goat.State
}
