package openapi

import (
	"context"
	"reflect"

	"github.com/goatx/goat"
)

type OpenAPIResponse[O AbstractOpenAPIEndpoint] struct {
	event O
}

func OpenAPISendTo[O AbstractOpenAPIEndpoint](ctx context.Context, target goat.AbstractStateMachine, event O) OpenAPIResponse[O] {
	goat.SendTo(ctx, target, event)
	return OpenAPIResponse[O]{event: event}
}

func NewOpenAPIServiceSpec[T goat.AbstractStateMachine](prototype T) *OpenAPIServiceSpec[T] {
	return &OpenAPIServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		endpoints:        []endpointMetadata{},
		schemas:          make(map[string]*schemaDefinition),
	}
}

func OnOpenAPIEndpoint[T goat.AbstractStateMachine, I AbstractOpenAPIEndpoint, O AbstractOpenAPIEndpoint](
	spec *OpenAPIServiceSpec[T],
	state goat.AbstractState,
	method string,
	path string,
	operationID string,
	requestEvent I,
	responseEvent O,
	handler func(context.Context, I, T) OpenAPIResponse[O],
) {
	requestTypeName := getEventTypeName(requestEvent)
	responseTypeName := getEventTypeName(responseEvent)

	metadata := endpointMetadata{
		Path:         path,
		Method:       method,
		OperationID:  operationID,
		RequestType:  requestTypeName,
		ResponseType: responseTypeName,
	}

	spec.addEndpoint(metadata)

	requestSchema := analyzeSchema(requestEvent)
	responseSchema := analyzeSchema(responseEvent)
	spec.addSchema(requestSchema)
	spec.addSchema(responseSchema)

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		response := handler(ctx, event, sm)
		_ = response
	}

	goat.OnEvent(spec.StateMachineSpec, state, requestEvent, wrappedHandler)
}

func analyzeSchema[S AbstractOpenAPIEndpoint](instance S) *schemaDefinition {
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
		if field.Type == reflect.TypeOf(OpenAPIEndpoint[goat.AbstractStateMachine, goat.AbstractStateMachine]{}) {
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

func mapGoFieldToOpenAPI(goType reflect.Type) (string, string, bool) {
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

func GenerateOpenAPI(opts GenerateOptions, specs ...AbstractOpenAPIServiceSpec) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./openapi"
	}
	if opts.Filename == "" {
		opts.Filename = "openapi.yaml"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}

	generator := newOpenAPIGenerator(opts)
	return generator.generateFromSpecs(specs...)
}

func getServiceTypeName[T goat.AbstractStateMachine](_ *goat.StateMachineSpec[T]) string {
	var zero T
	return getTypeName(zero)
}

func getEventTypeName[E AbstractOpenAPIEndpoint](event E) string {
	return getTypeName(event)
}

func getTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
