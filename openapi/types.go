package openapi

import (
	"github.com/goatx/goat"
)

// StatusCode represents an HTTP status code
type StatusCode int

// HTTP status codes
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

type AbstractOpenAPISchema interface {
	isOpenAPISchema() bool
	goat.AbstractEvent
}

type OpenAPISchema[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	// this is needed to make OpenAPISchema copyable
	_ rune
}

func (*OpenAPISchema[Sender, Recipient]) isOpenAPISchema() bool {
	return true
}

// OpenAPIRequest represents a request schema in OpenAPI specification
type OpenAPIRequest[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	// this is needed to make OpenAPIRequest copyable
	_ rune
}

func (*OpenAPIRequest[Sender, Recipient]) isOpenAPISchema() bool {
	return true
}

// OpenAPIResponse represents a response schema in OpenAPI specification with status code
type OpenAPIResponse[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	StatusCode int
	// this is needed to make OpenAPIResponse copyable
	_ rune
}

func (*OpenAPIResponse[Sender, Recipient]) isOpenAPISchema() bool {
	return true
}

type AbstractOpenAPIServiceSpec interface {
	isOpenAPIServiceSpec() bool
	GetEndpoints() []endpointMetadata
	GetSchemas() map[string]*schemaDefinition
}

type OpenAPIServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	endpoints []endpointMetadata
	schemas   map[string]*schemaDefinition
}

func (*OpenAPIServiceSpec[T]) isOpenAPIServiceSpec() bool {
	return true
}

func (os *OpenAPIServiceSpec[T]) GetEndpoints() []endpointMetadata {
	return os.endpoints
}

func (os *OpenAPIServiceSpec[T]) GetSchemas() map[string]*schemaDefinition {
	return os.schemas
}

//nolint:gocritic // metadata is passed by value for consistency with protobuf package
func (os *OpenAPIServiceSpec[T]) addEndpoint(metadata endpointMetadata) {
	os.endpoints = append(os.endpoints, metadata)
}

func (os *OpenAPIServiceSpec[T]) addSchema(schema *schemaDefinition) {
	if os.schemas == nil {
		os.schemas = make(map[string]*schemaDefinition)
	}
	os.schemas[schema.Name] = schema
}

type endpointMetadata struct {
	Path         string
	Method       string
	OperationID  string
	RequestType  string
	ResponseType string
	StatusCode   StatusCode
}

type schemaDefinition struct {
	Name   string
	Fields []schemaField
}

type schemaField struct {
	Name     string
	Type     string
	Format   string
	IsArray  bool
	Required bool
}
