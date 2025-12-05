package openapi

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/strcase"
	"github.com/goatx/goat/internal/typeutil"
)

// Response represents a response that will be sent back in an OpenAPI RPC handler.
// You must create this by calling SendTo with your response message.
type Response[O AbstractSchema] interface {
	isResponse()
}

type responseImpl[O AbstractSchema] struct {
	event O
}

func (responseImpl[O]) isResponse() {}

// SendTo sends an OpenAPI response event to the target state machine and returns a Response.
// This is the only way to create a Response, enforcing that all responses go through the proper event system.
//
// Type parameters:
//   - O: The response schema type (must implement AbstractSchema)
//
// Parameters:
//   - ctx: The context for the event
//   - target: The target state machine to send the event to
//   - event: The response event to send
//
// Returns:
//   - Response[O]: A sealed response type that can only be created through this function
func SendTo[O AbstractSchema](ctx context.Context, target goat.AbstractStateMachine, event O) Response[O] {
	goat.SendTo(ctx, target, event)
	return responseImpl[O]{event: event}
}

// RequestOption is a functional option for configuring an OpenAPI request handler.
type RequestOption func(*requestConfig)

type requestConfig struct {
	operationID    string
	statusCode     StatusCode
	isBodyOptional bool
}

// WithOperationID sets a custom operation ID for the endpoint.
// If the operationID is empty, the operationId field will be omitted from the generated OpenAPI spec.
func WithOperationID(id string) RequestOption {
	return func(c *requestConfig) {
		c.operationID = id
	}
}

// WithStatusCode sets a custom HTTP status code for the endpoint response.
// The default status code is 200 (StatusOK) if not specified.
func WithStatusCode(code StatusCode) RequestOption {
	return func(c *requestConfig) {
		c.statusCode = code
	}
}

// WithRequestBodyOptional marks the request body as optional in the generated OpenAPI spec.
func WithRequestBodyOptional() RequestOption {
	return func(c *requestConfig) {
		c.isBodyOptional = true
	}
}

// NewServiceSpec creates a new OpenAPI service specification for the given
// state machine prototype. The prototype defines which state machine implementation
// will receive requests and how its states will be exposed via OpenAPI.
//
// Parameters:
//   - prototype: The state machine instance used as the template for the service
//
// Returns a specification that can be configured with states and handlers.
//
// Example:
//
//	spec := openapi.NewServiceSpec(&UserStateMachine{})
//	openapi.OnRequest(spec, UserState{}, HTTPMethodPost, "/users", handleCreateUser)
func NewServiceSpec[T goat.AbstractStateMachine](prototype T) *ServiceSpec[T] {
	return &ServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		endpoints:        []endpointMetadata{},
		schemas:          make(map[string]*schemaDefinition),
	}
}

// OnRequest registers an OpenAPI endpoint handler with the given specification.
//
// Type parameters:
//   - T: The state machine type
//   - I: The request schema type (must implement AbstractSchema)
//   - O: The response schema type (must implement AbstractSchema)
//
// Parameters:
//   - spec: The OpenAPI service specification to register the endpoint with
//   - state: The state in which this endpoint is active
//   - method: The HTTP method (e.g., HTTPMethodGet, HTTPMethodPost)
//   - path: The URL path for the endpoint (e.g., "/users/{id}")
//   - handler: The function that handles requests for this endpoint
//   - opts: Optional configuration options (WithOperationID, WithStatusCode)
func OnRequest[T goat.AbstractStateMachine, I AbstractSchema, O AbstractSchema](
	spec *ServiceSpec[T],
	state goat.AbstractState,
	method HTTPMethod,
	path string,
	handler func(context.Context, I, T) Response[O],
	opts ...RequestOption,
) {
	if method == nil {
		panic("http method must not be nil")
	}

	config := &requestConfig{
		operationID:    "",
		statusCode:     StatusOK,
		isBodyOptional: false,
	}

	for _, opt := range opts {
		opt(config)
	}

	requestEvent := newSchemaPrototype[I]()
	responseEvent := newSchemaPrototype[O]()

	requestTypeName := getEventTypeName(requestEvent)
	responseTypeName := getEventTypeName(responseEvent)

	metadata := endpointMetadata{
		Path:           path,
		Method:         method,
		OperationID:    config.operationID,
		RequestType:    requestTypeName,
		ResponseType:   responseTypeName,
		StatusCode:     config.statusCode,
		IsBodyOptional: config.isBodyOptional,
	}

	spec.addEndpoint(&metadata)

	requestSchema := analyzeSchema(requestEvent)
	responseSchema := analyzeSchema(responseEvent)
	spec.addSchema(requestSchema)
	spec.addSchema(responseSchema)

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		_ = handler(ctx, event, sm)
	}

	goat.OnEvent(spec.StateMachineSpec, state, wrappedHandler)
}

func newSchemaPrototype[T AbstractSchema]() T {
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
			panic(fmt.Sprintf("type %s does not implement AbstractSchema", msgType))
		}
		return msg
	}

	value := reflect.New(msgType).Elem().Interface()
	msg, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("type %s does not implement AbstractSchema", msgType))
	}
	return msg
}

func analyzeSchema[S AbstractSchema](instance S) *schemaDefinition {
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
		if field.Type == reflect.TypeOf(Schema[goat.AbstractStateMachine, goat.AbstractStateMachine]{}) {
			continue
		}
		if field.Name == "_" {
			continue
		}

		openAPIType, format, isArray := mapGoField(field.Type)
		if openAPIType == "" {
			continue
		}

		fieldName, paramType, isRequired := parseField(&field)

		newField := schemaField{
			Name:      fieldName,
			Type:      openAPIType,
			Format:    format,
			IsArray:   isArray,
			Required:  isRequired,
			ParamType: paramType,
		}

		fields = append(fields, newField)
	}

	return &schemaDefinition{
		Name:   schemaType.Name(),
		Fields: fields,
	}
}

func mapGoField(goType reflect.Type) (typeName, format string, isArray bool) {
	if goType.Kind() == reflect.Slice {
		elemType, format, _ := mapGoField(goType.Elem())
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

// GenerateOptions contains configuration options for OpenAPI spec generation.
type GenerateOptions struct {
	// OutputDir is the directory where the OpenAPI spec file will be written.
	// Defaults to "./openapi" if not specified.
	OutputDir string

	// Title is the title of the API (required).
	// This appears in the "info.title" field of the OpenAPI spec.
	Title string

	// Version is the version of the API.
	// Defaults to "1.0.0" if not specified.
	Version string

	// Filename is the name of the generated OpenAPI spec file.
	// Defaults to "openapi.yaml" if not specified.
	Filename string
}

// Generate generates an OpenAPI 3.0 specification file from one or more service specifications.
// It analyzes the registered endpoints and schemas, and writes a YAML file to the specified output directory.
//
// Parameters:
//   - opts: Configuration options for the generation (Title is required)
//   - specs: One or more OpenAPI service specifications to include in the generated spec
//
// Returns:
//   - error: An error if Title is empty, or if file creation/writing fails
func Generate(opts *GenerateOptions, specs ...AbstractServiceSpec) error {
	if opts.Title == "" {
		return fmt.Errorf("title is required in GenerateOptions")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "./openapi"
	}
	if opts.Filename == "" {
		opts.Filename = "openapi.yaml"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}

	generator := newGenerator(*opts)
	return generator.generateFromSpecs(specs...)
}

func getEventTypeName[E AbstractSchema](event E) string {
	return typeutil.Name(event)
}

func parseField(field *reflect.StructField) (fieldName string, paramType parameterType, isRequired bool) {
	fieldName = strcase.ToCamelCase(field.Name)

	tag := field.Tag.Get("openapi")
	if tag == "" {
		return fieldName, parameterTypeNone, false
	}

	parts := strings.Split(tag, ",")

	hasDefinition := false

	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}

		if part == "required" {
			isRequired = true
			continue
		}

		if strings.Contains(part, "=") {
			kv := strings.Split(part, "=")
			if len(kv) != 2 {
				log.Printf("[WARNING] openapi: ignoring invalid openapi tag %q on field %s: expected parameter definition", tag, field.Name)
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if key == "" || value == "" {
				log.Printf("[WARNING] openapi: ignoring invalid openapi tag %q on field %s: expected parameter type and name", tag, field.Name)
				continue
			}
			if hasDefinition {
				log.Printf("[WARNING] openapi: ignoring extra parameter definition %q on field %s", part, field.Name)
				continue
			}
			pt := toParameterType(key)
			if pt == parameterTypeInvalid {
				log.Printf("[WARNING] openapi: unsupported parameter type %q in tag %q on field %s", key, tag, field.Name)
				continue
			}
			paramType = pt
			fieldName = value
			hasDefinition = true
			continue
		}

		if hasDefinition {
			log.Printf("[WARNING] openapi: ignoring unsupported modifier %q on field %s", part, field.Name)
			continue
		}

		pt := toParameterType(part)
		if pt == parameterTypeInvalid {
			log.Printf("[WARNING] openapi: unsupported parameter type %q in tag %q on field %s", part, tag, field.Name)
			continue
		}
		paramType = pt
		fieldName = strcase.ToSnakeCase(field.Name)
		hasDefinition = true
	}

	if paramType == parameterTypePath {
		isRequired = true
	}

	return fieldName, paramType, isRequired
}

func toParameterType(value string) parameterType {
	switch value {
	case "path":
		return parameterTypePath
	case "query":
		return parameterTypeQuery
	case "header":
		return parameterTypeHeader
	default:
		return parameterTypeInvalid
	}
}
