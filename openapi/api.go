package openapi

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goatx/goat"
)

type OpenAPIResponse[O AbstractOpenAPISchema] struct {
	event O
}

func OpenAPISendTo[O AbstractOpenAPISchema](ctx context.Context, target goat.AbstractStateMachine, event O) OpenAPIResponse[O] {
	goat.SendTo(ctx, target, event)
	return OpenAPIResponse[O]{event: event}
}

// RequestOption is a functional option for configuring OnOpenAPIRequest
type RequestOption func(*requestConfig)

type requestConfig struct {
	operationID string
	statusCode  StatusCode
}

// WithOperationID sets a custom operation ID for the endpoint
func WithOperationID(id string) RequestOption {
	return func(c *requestConfig) {
		c.operationID = id
	}
}

// WithStatusCode sets a custom status code for the endpoint
func WithStatusCode(code StatusCode) RequestOption {
	return func(c *requestConfig) {
		c.statusCode = code
	}
}

func NewOpenAPIServiceSpec[T goat.AbstractStateMachine](prototype T) *OpenAPIServiceSpec[T] {
	return &OpenAPIServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		endpoints:        []endpointMetadata{},
		schemas:          make(map[string]*schemaDefinition),
	}
}

func OnOpenAPIRequest[T goat.AbstractStateMachine, I AbstractOpenAPISchema, O AbstractOpenAPISchema](
	spec *OpenAPIServiceSpec[T],
	state goat.AbstractState,
	method string,
	path string,
	handler func(context.Context, I, T) OpenAPIResponse[O],
	opts ...RequestOption,
) {
	// Apply default configuration
	config := &requestConfig{
		operationID: "", // Empty means don't generate operationID field
		statusCode:  StatusOK,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(config)
	}

	requestEvent := newOpenAPISchemaPrototype[I]()
	responseEvent := newOpenAPISchemaPrototype[O]()

	requestTypeName := getEventTypeName(requestEvent)
	responseTypeName := getEventTypeName(responseEvent)

	metadata := endpointMetadata{
		Path:         path,
		Method:       method,
		OperationID:  config.operationID,
		RequestType:  requestTypeName,
		ResponseType: responseTypeName,
		StatusCode:   config.statusCode,
	}

	spec.addEndpoint(metadata)

	requestSchema := analyzeSchema(requestEvent)
	responseSchema := analyzeSchema(responseEvent)
	spec.addSchema(requestSchema)
	spec.addSchema(responseSchema)

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		_ = handler(ctx, event, sm)
	}

	goat.OnEvent(spec.StateMachineSpec, state, wrappedHandler)
}

func newOpenAPISchemaPrototype[T AbstractOpenAPISchema]() T {
	var zero T
	msgType := reflect.TypeOf(zero)
	if msgType == nil {
		msgType = reflect.TypeFor[T]()
	}

	if msgType.Kind() == reflect.Interface {
		panic(fmt.Sprintf("cannot use interface type %s as openapi schema type parameter; use a concrete schema type instead", msgType))
	}

	if msgType.Kind() == reflect.Pointer {
		elem := msgType.Elem()
		if elem.Kind() == reflect.Interface {
			panic(fmt.Sprintf("cannot use interface type %s as openapi schema type parameter; use a concrete schema type instead", elem))
		}

		prototype := reflect.New(elem).Interface()
		msg, ok := prototype.(T)
		if !ok {
			panic(fmt.Sprintf("type %s does not implement AbstractOpenAPISchema", msgType))
		}
		return msg
	}

	value := reflect.New(msgType).Elem().Interface()
	msg, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("type %s does not implement AbstractOpenAPISchema", msgType))
	}
	return msg
}

func analyzeSchema[S AbstractOpenAPISchema](instance S) *schemaDefinition {
	schemaType := reflect.TypeOf(instance)
	if schemaType.Kind() == reflect.Ptr {
		schemaType = schemaType.Elem()
	}

	var fields []schemaField

	for i := 0; i < schemaType.NumField(); i++ {
		field := schemaType.Field(i)

		if !field.IsExported() {
			continue
		}
		if field.Type == reflect.TypeOf(OpenAPISchema[goat.AbstractStateMachine, goat.AbstractStateMachine]{}) {
			continue
		}
		if isGoatEventType(field.Type) {
			continue
		}
		if field.Name == "_" {
			continue
		}

		openAPIType, format, isArray := mapGoFieldToOpenAPI(field.Type)
		if openAPIType == "" {
			continue
		}

		fields = append(fields, schemaField{
			Name:     field.Name,
			Type:     openAPIType,
			Format:   format,
			IsArray:  isArray,
			Required: true,
		})
	}

	return &schemaDefinition{
		Name:   schemaType.Name(),
		Fields: fields,
	}
}

func isGoatEventType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct && t.Name() == "Event" && t.PkgPath() == "github.com/goatx/goat"
}

func mapGoFieldToOpenAPI(goType reflect.Type) (typeName, format string, isArray bool) {
	if goType.Kind() == reflect.Slice {
		elemType, format, _ := mapGoFieldToOpenAPI(goType.Elem())
		return elemType, format, true
	}

	switch goType.Kind() {
	case reflect.String:
		return "string", "", false
	case reflect.Bool:
		return "boolean", "", false
	case reflect.Int32:
		return "integer", "int32", false
	case reflect.Int64, reflect.Int:
		return "integer", "int64", false
	case reflect.Float32:
		return "number", "float", false
	case reflect.Float64:
		return "number", "double", false
	default:
		return "", "", false
	}
}

type GenerateOptions struct {
	OutputDir   string
	Title       string
	Version     string
	Description string
	Filename    string
}

func GenerateOpenAPI(opts *GenerateOptions, specs ...AbstractOpenAPIServiceSpec) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./openapi"
	}
	if opts.Filename == "" {
		opts.Filename = "openapi.yaml"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}

	generator := newOpenAPIGenerator(*opts)
	return generator.generateFromSpecs(specs...)
}

func getEventTypeName[E AbstractOpenAPISchema](event E) string {
	return getTypeName(event)
}

func getTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
