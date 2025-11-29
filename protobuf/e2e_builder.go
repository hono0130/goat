package protobuf

import (
	"fmt"
	"reflect"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/e2egen"
	"github.com/goatx/goat/internal/strcase"
)

func buildTestSuite(opts E2ETestOptions) (e2egen.TestSuite, error) {
	suite := e2egen.TestSuite{
		PackageName: opts.PackageName,
	}

	for si, svc := range opts.Services {
		serviceName := svc.Spec.GetServiceName()
		clientVarName := strcase.ToSnakeCase(serviceName) + "Client"

		var methods []e2egen.MethodTestSuite

		for mi, method := range svc.Methods {
			var cases []e2egen.TestCase

			for ii, input := range method.Inputs {
				output, err := executeHandler(svc.Spec, method.MethodName, input)
				if err != nil {
					return e2egen.TestSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to execute handler: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				inputData, err := serializeMessage(input)
				if err != nil {
					return e2egen.TestSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize input: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				outputData, err := serializeMessage(output)
				if err != nil {
					return e2egen.TestSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize output: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				cases = append(cases, e2egen.TestCase{
					Name:       fmt.Sprintf("case_%d", ii),
					InputType:  getTypeName(input),
					Input:      inputData,
					OutputType: getTypeName(output),
					Output:     outputData,
				})
			}

			methods = append(methods, e2egen.MethodTestSuite{
				MethodName: method.MethodName,
				TestCases:  cases,
			})
		}

		suite.Services = append(suite.Services, e2egen.ServiceTestSuite{
			ServiceName:    serviceName,
			ServicePackage: svc.ServicePackage,
			ClientVarName:  clientVarName,
			Methods:        methods,
		})
	}

	return suite, nil
}

func executeHandler(spec AbstractProtobufServiceSpec, methodName string, input AbstractProtobufMessage) (AbstractProtobufMessage, error) {
	handlers := spec.GetHandlers()
	handler, ok := handlers[methodName]
	if !ok {
		return nil, fmt.Errorf("no handler found for method %s", methodName)
	}

	instance, err := spec.NewStateMachineInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine instance: %w", err)
	}

	ctx := goat.NewHandlerContext(instance)

	handlerValue := reflect.ValueOf(handler)
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(input),
		reflect.ValueOf(instance),
	})

	response := results[0]
	eventResults := response.MethodByName("GetEvent").Call(nil)

	return eventResults[0].Interface().(AbstractProtobufMessage), nil
}
