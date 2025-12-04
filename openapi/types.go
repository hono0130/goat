package openapi

import (
	"github.com/goatx/goat"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusCreated             StatusCode = 201
	StatusAccepted            StatusCode = 202
	StatusNoContent           StatusCode = 204
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusConflict            StatusCode = 409
	StatusInternalServerError StatusCode = 500
	StatusServiceUnavailable  StatusCode = 503
)

type AbstractSchema interface {
	isSchema() bool
	goat.AbstractEvent
}

type HTTPMethod interface {
	String() string
	httpMethod()
}

type httpMethodValue string

func (m httpMethodValue) String() string {
	return string(m)
}

func (httpMethodValue) httpMethod() {}

var (
	HTTPMethodGet    HTTPMethod = httpMethodValue("GET")
	HTTPMethodPost   HTTPMethod = httpMethodValue("POST")
	HTTPMethodPut    HTTPMethod = httpMethodValue("PUT")
	HTTPMethodDelete HTTPMethod = httpMethodValue("DELETE")
)

type Schema[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	// this is needed to make Schema copyable
	_ rune
}

func (*Schema[Sender, Recipient]) isSchema() bool {
	return true
}

type AbstractServiceSpec interface {
	isServiceSpec() bool
	getEndpoints() []endpointMetadata
	getSchemas() map[string]*schemaDefinition
}

type ServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	endpoints []endpointMetadata
	schemas   map[string]*schemaDefinition
}

func (*ServiceSpec[T]) isServiceSpec() bool {
	return true
}

func (os *ServiceSpec[T]) getEndpoints() []endpointMetadata {
	return os.endpoints
}

func (os *ServiceSpec[T]) getSchemas() map[string]*schemaDefinition {
	return os.schemas
}

func (os *ServiceSpec[T]) addEndpoint(metadata *endpointMetadata) {
	os.endpoints = append(os.endpoints, *metadata)
}

func (os *ServiceSpec[T]) addSchema(schema *schemaDefinition) {
	if os.schemas == nil {
		os.schemas = make(map[string]*schemaDefinition)
	}
	os.schemas[schema.Name] = schema
}

type endpointMetadata struct {
	Path           string
	Method         HTTPMethod
	OperationID    string
	RequestType    string
	ResponseType   string
	StatusCode     StatusCode
	IsBodyOptional bool
}

type schemaDefinition struct {
	Name   string
	Fields []schemaField
}

type schemaField struct {
	Name      string
	Type      string
	Format    string
	IsArray   bool
	Required  bool
	ParamType parameterType
}

type parameterType string

const (
	parameterTypeNone    parameterType = ""
	parameterTypePath    parameterType = "path"
	parameterTypeQuery   parameterType = "query"
	parameterTypeHeader  parameterType = "header"
	parameterTypeInvalid parameterType = ""
)

func (p parameterType) String() string {
	switch p {
	case parameterTypePath:
		return "path"
	case parameterTypeQuery:
		return "query"
	case parameterTypeHeader:
		return "header"
	default:
		return ""
	}
}
