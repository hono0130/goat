package protobuf

import (
	"context"
	"reflect"

	"github.com/goatx/goat"
)

type ProtobufResponse[O AbstractProtobufMessage] struct {
	event O
}

func ProtobufSendTo[O AbstractProtobufMessage](ctx context.Context, target goat.AbstractStateMachine, event O) ProtobufResponse[O] {
	goat.SendTo(ctx, target, event)
	return ProtobufResponse[O]{event: event}
}

func NewProtobufServiceSpec[T goat.AbstractStateMachine](prototype T) *ProtobufServiceSpec[T] {
	return &ProtobufServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		rpcMethods:       []rpcMethod{},
		messages:         make(map[string]*protoMessage),
	}
}

func OnProtobufMessage[T goat.AbstractStateMachine, I AbstractProtobufMessage, O AbstractProtobufMessage](
	spec *ProtobufServiceSpec[T],
	state goat.AbstractState,
	methodName string,
	inputEvent I,
	outputEvent O,
	handler func(context.Context, I, T) ProtobufResponse[O],
) {
	serviceTypeName := getServiceTypeName(spec.StateMachineSpec)
	inputTypeName := getEventTypeName(inputEvent)
	outputTypeName := getEventTypeName(outputEvent)

	metadata := rpcMethod{
		ServiceType: serviceTypeName,
		MethodName:  methodName,
		InputType:   inputTypeName,
		OutputType:  outputTypeName,
	}

	spec.addRPCMethod(metadata)

	inputMsg := analyzeMessage(inputEvent)
	outputMsg := analyzeMessage(outputEvent)
	spec.addMessage(inputMsg)
	spec.addMessage(outputMsg)

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		response := handler(ctx, event, sm)
		_ = response
	}

	goat.OnEvent(spec.StateMachineSpec, state, inputEvent, wrappedHandler)
}

func analyzeMessage[M AbstractProtobufMessage](instance M) *protoMessage {
	msgType := reflect.TypeOf(instance)
	if msgType.Kind() == reflect.Ptr {
		msgType = msgType.Elem()
	}

	var fields []protoField
	fieldNum := 1

	protobufMessageType := reflect.TypeOf(ProtobufMessage{})
	for i := 0; i < msgType.NumField(); i++ {
		field := msgType.Field(i)

		if !field.IsExported() {
			continue
		}
		if field.Type == protobufMessageType {
			continue
		}
		if isGoatEventType(field.Type) {
			continue
		}
		if field.Name == "_" {
			continue
		}

		protoType, isRepeated := mapGoFieldToProto(field.Type)
		if protoType == "" {
			continue
		}

		fields = append(fields, protoField{
			Name:       field.Name,
			Type:       protoType,
			Number:     fieldNum,
			IsRepeated: isRepeated,
		})
		fieldNum++
	}

	return &protoMessage{
		Name:   msgType.Name(),
		Fields: fields,
	}
}

func isGoatEventType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct && t.Name() == "Event" && t.PkgPath() == "github.com/goatx/goat"
}

func mapGoFieldToProto(goType reflect.Type) (string, bool) {
	if goType.Kind() == reflect.Slice {
		elemType, _ := mapGoFieldToProto(goType.Elem())
		return elemType, true
	}

	switch goType.Kind() {
	case reflect.String:
		return "string", false
	case reflect.Bool:
		return "bool", false
	case reflect.Int32:
		return "int32", false
	case reflect.Int64, reflect.Int:
		return "int64", false
	case reflect.Float32:
		return "float", false
	case reflect.Float64:
		return "double", false
	default:
		return "", false
	}
}

type GenerateOptions struct {
	OutputDir   string
	PackageName string
	GoPackage   string
	Filename    string
}

func GenerateProtobuf(opts GenerateOptions, specs ...AbstractProtobufServiceSpec) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./proto"
	}
	if opts.Filename == "" {
		opts.Filename = "generated.proto"
	}

	generator := newProtobufGenerator(opts)
	return generator.generateFromSpecs(specs...)
}

func getServiceTypeName[T goat.AbstractStateMachine](_ *goat.StateMachineSpec[T]) string {
	var zero T
	return getTypeName(zero)
}

func getEventTypeName[E AbstractProtobufMessage](event E) string {
	return getTypeName(event)
}

func getTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
