package protobuf

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goatx/goat"
)

// buildTestSuite builds an intermediate representation (testSuite) from protobuf E2ETestOptions.
// This function is responsible for:
// 1. Executing handlers to calculate expected outputs
// 2. Serializing input/output messages
// 3. Constructing the protocol-agnostic intermediate representation
func buildTestSuite(opts E2ETestOptions) (testSuite, error) {
	suite := testSuite{
		packageName: opts.PackageName,
	}

	for si, svc := range opts.Services {
		serviceName := svc.Spec.GetServiceName()
		clientVarName := toSnakeCase(serviceName) + "Client"

		var methods []methodTestSuite

		for mi, method := range svc.Methods {
			var cases []testCase

			for ii, input := range method.Inputs {
				// Execute handler to get expected output
				output, err := executeHandler(svc.Spec, method.MethodName, input)
				if err != nil {
					return testSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to execute handler: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				// Serialize input and output
				inputData, err := serializeMessage(input)
				if err != nil {
					return testSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize input: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				outputData, err := serializeMessage(output)
				if err != nil {
					return testSuite{}, fmt.Errorf("service %d (%s) method %d (%s) input %d: failed to serialize output: %w",
						si, serviceName, mi, method.MethodName, ii, err)
				}

				cases = append(cases, testCase{
					name:       fmt.Sprintf("case_%d", ii),
					inputType:  getTypeName(input),
					input:      inputData,
					outputType: getTypeName(output),
					output:     outputData,
				})
			}

			methods = append(methods, methodTestSuite{
				methodName: method.MethodName,
				testCases:  cases,
			})
		}

		suite.services = append(suite.services, serviceTestSuite{
			serviceName:    serviceName,
			servicePackage: svc.ServicePackage,
			clientVarName:  clientVarName,
			methods:        methods,
		})
	}

	return suite, nil
}

// executeHandler executes a handler for the given method and input, returning the output event.
func executeHandler(spec AbstractProtobufServiceSpec, methodName string, input AbstractProtobufMessage) (AbstractProtobufMessage, error) {
	// Get handler for this method
	handlers := spec.GetHandlers()
	handler, ok := handlers[methodName]
	if !ok {
		return nil, fmt.Errorf("no handler found for method %s", methodName)
	}

	// Create state machine instance
	instance, err := spec.NewStateMachineInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine instance: %w", err)
	}

	// Create context for handler execution
	ctx := goat.NewHandlerContext(instance)

	// Call handler using reflection: handler(ctx, input, instance)
	handlerValue := reflect.ValueOf(handler)
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(input),
		reflect.ValueOf(instance),
	})

	// Extract event from ProtobufResponse using GetEvent()
	response := results[0]
	eventResults := response.MethodByName("GetEvent").Call(nil)

	return eventResults[0].Interface().(AbstractProtobufMessage), nil
}

// toSnakeCase converts a PascalCase or camelCase string to snake_case.
func toSnakeCase(name string) string {
	var result strings.Builder

	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prevChar := rune(name[i-1])
				if prevChar >= 'a' && prevChar <= 'z' {
					result.WriteRune('_')
				} else if i < len(name)-1 {
					nextChar := rune(name[i+1])
					if nextChar >= 'a' && nextChar <= 'z' {
						result.WriteRune('_')
					}
				}
			}
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
