package openapi

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
	Schema[*TestService1, *TestService1]
	Data string `openapi:"required"`
}

type TestResponse1 struct {
	Schema[*TestService1, *TestService1]
	Result string `openapi:"required"`
}

type TestRequest2 struct {
	Schema[*TestService1, *TestService1]
	Info string `openapi:"required"`
}

type TestResponse2 struct {
	Schema[*TestService1, *TestService1]
	Value string `openapi:"required"`
}

type TestIdleState struct {
	goat.State
}
