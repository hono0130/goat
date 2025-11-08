package openapi

import (
	"reflect"
	"sort"
)

type schemaAnalyzer struct {
	processedTypes map[reflect.Type]bool
}

func newSchemaAnalyzer() *schemaAnalyzer {
	return &schemaAnalyzer{
		processedTypes: make(map[reflect.Type]bool),
	}
}

type pathOperation struct {
	Method      string
	OperationID string
	RequestRef  string
	ResponseRef string
}

type pathDefinition struct {
	Path       string
	Operations []pathOperation
}

type openAPIDefinitions struct {
	Schemas []*schemaDefinition
	Paths   []*pathDefinition
}

func (a *schemaAnalyzer) analyzeSpecs(specs ...AbstractOpenAPIServiceSpec) *openAPIDefinitions {
	definitions := &openAPIDefinitions{
		Schemas: []*schemaDefinition{},
		Paths:   []*pathDefinition{},
	}

	a.processedTypes = make(map[reflect.Type]bool)

	pathMap := make(map[string]*pathDefinition)

	for _, spec := range specs {
		endpoints := spec.GetEndpoints()

		for _, endpoint := range endpoints {
			if pathMap[endpoint.Path] == nil {
				pathMap[endpoint.Path] = &pathDefinition{
					Path:       endpoint.Path,
					Operations: []pathOperation{},
				}
			}

			operation := pathOperation{
				Method:      endpoint.Method,
				OperationID: endpoint.OperationID,
				RequestRef:  endpoint.RequestType,
				ResponseRef: endpoint.ResponseType,
			}
			pathMap[endpoint.Path].Operations = append(pathMap[endpoint.Path].Operations, operation)
		}

		schemas := make([]*schemaDefinition, 0, len(spec.GetSchemas()))
		for _, schema := range spec.GetSchemas() {
			schemas = append(schemas, schema)
		}
		sort.Slice(schemas, func(i, j int) bool {
			return schemas[i].Name < schemas[j].Name
		})

		definitions.Schemas = append(definitions.Schemas, schemas...)
	}

	for _, path := range pathMap {
		definitions.Paths = append(definitions.Paths, path)
	}
	sort.Slice(definitions.Paths, func(i, j int) bool {
		return definitions.Paths[i].Path < definitions.Paths[j].Path
	})

	return definitions
}
