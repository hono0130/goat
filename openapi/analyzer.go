package openapi

import (
	"sort"
)

type schemaAnalyzer struct{}

func newSchemaAnalyzer() *schemaAnalyzer {
	return &schemaAnalyzer{}
}

type pathOperation struct {
	Method         HTTPMethod
	OperationID    string
	RequestRef     string
	RequestSchema  *schemaDefinition
	Responses      []operationResponse
	IsBodyOptional bool
}

type operationResponse struct {
	StatusCode  StatusCode
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

func (*schemaAnalyzer) analyzeSpecs(specs ...AbstractOpenAPIServiceSpec) *openAPIDefinitions {
	definitions := &openAPIDefinitions{}

	operations := make(map[string]map[string]*pathOperation)

	for _, spec := range specs {
		specSchemas := spec.getSchemas()
		endpoints := spec.getEndpoints()

		for _, endpoint := range endpoints {
			methods := operations[endpoint.Path]
			if methods == nil {
				methods = make(map[string]*pathOperation)
				operations[endpoint.Path] = methods
			}

			op := methods[endpoint.Method.String()]
			if op == nil {
				op = &pathOperation{
					Method:         endpoint.Method,
					OperationID:    endpoint.OperationID,
					RequestRef:     endpoint.RequestType,
					RequestSchema:  specSchemas[endpoint.RequestType],
					IsBodyOptional: endpoint.IsBodyOptional,
				}
				methods[endpoint.Method.String()] = op
			}

			op.Responses = append(op.Responses, operationResponse{
				StatusCode:  endpoint.StatusCode,
				ResponseRef: endpoint.ResponseType,
			})
		}

		schemas := make([]*schemaDefinition, 0, len(specSchemas))
		for _, schema := range specSchemas {
			schemas = append(schemas, schema)
		}
		sort.Slice(schemas, func(i, j int) bool {
			return schemas[i].Name < schemas[j].Name
		})

		definitions.Schemas = append(definitions.Schemas, schemas...)
	}

	for path, methods := range operations {
		pathDef := &pathDefinition{
			Path: path,
		}
		for _, op := range methods {
			pathDef.Operations = append(pathDef.Operations, *op)
		}
		definitions.Paths = append(definitions.Paths, pathDef)
	}
	sort.Slice(definitions.Paths, func(i, j int) bool {
		return definitions.Paths[i].Path < definitions.Paths[j].Path
	})

	return definitions
}
