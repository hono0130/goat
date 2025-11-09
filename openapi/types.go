package openapi

import (
	"github.com/goatx/goat"
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
