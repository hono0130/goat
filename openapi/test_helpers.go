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
	OpenAPISchema[*TestService1, *TestService1]
	Data string
}

type TestResponse1 struct {
	OpenAPISchema[*TestService1, *TestService1]
	Result string
}

type TestRequest2 struct {
	OpenAPISchema[*TestService1, *TestService1]
	Info string
}

type TestResponse2 struct {
	OpenAPISchema[*TestService1, *TestService1]
	Value string
}

type TestRequest3 struct {
	OpenAPISchema[*TestService1, *TestService1]
	Input string
}

type TestResponse3 struct {
	OpenAPISchema[*TestService1, *TestService1]
	Output string
}

type TestIdleState struct {
	goat.State
}
