package protobuf

import (
	"reflect"
	"sort"
)

type typeAnalyzer struct {
	processedTypes map[reflect.Type]bool
}

func newTypeAnalyzer() *typeAnalyzer {
	return &typeAnalyzer{
		processedTypes: make(map[reflect.Type]bool),
	}
}

type protoService struct {
	Name    string
	Methods []protoMethod
}

type protoMethod struct {
	Name       string
	InputType  string
	OutputType string
}

type protoDefinitions struct {
	Messages []*protoMessage
	Services []*protoService
}

func (a *typeAnalyzer) analyzeSpecs(specs ...AbstractProtobufServiceSpec) *protoDefinitions {
	definitions := &protoDefinitions{
		Messages: []*protoMessage{},
		Services: []*protoService{},
	}

	a.processedTypes = make(map[reflect.Type]bool)

	for _, spec := range specs {
		service := a.analyzeServiceSpecInterface(spec)
		definitions.Services = append(definitions.Services, service)

		messages := make([]*protoMessage, 0, len(spec.GetMessages()))
		for _, message := range spec.GetMessages() {
			messages = append(messages, message)
		}
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Name < messages[j].Name
		})

		definitions.Messages = append(definitions.Messages, messages...)
	}

	return definitions
}

func (*typeAnalyzer) analyzeServiceSpecInterface(spec AbstractProtobufServiceSpec) *protoService {
	rpcMethods := spec.GetRPCMethods()

	serviceName := ""
	if len(rpcMethods) > 0 {
		serviceName = rpcMethods[0].ServiceType
	}

	service := &protoService{
		Name:    serviceName,
		Methods: []protoMethod{},
	}

	for _, metadata := range rpcMethods {
		method := protoMethod{
			Name:       metadata.MethodName,
			InputType:  metadata.InputType,
			OutputType: metadata.OutputType,
		}
		service.Methods = append(service.Methods, method)
	}

	return service
}
