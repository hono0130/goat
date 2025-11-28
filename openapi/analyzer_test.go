package openapi

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSchemaAnalyzer_analyzeSpecs(t *testing.T) {
	tests := []struct {
		name    string
		specs   []AbstractOpenAPIServiceSpec
		want    *openAPIDefinitions
		wantErr bool
	}{
		{
			name:  "empty specs returns empty definitions",
			specs: []AbstractOpenAPIServiceSpec{},
			want: &openAPIDefinitions{
				Schemas: []*schemaDefinition{},
				Paths:   []*pathDefinition{},
			},
			wantErr: false,
		},
		{
			name: "single spec with one endpoint",
			specs: []AbstractOpenAPIServiceSpec{
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest(spec, state, HTTPMethodPost, "/test",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
							return OpenAPISendTo(ctx, sm, &TestResponse1{Result: "test"})
						},
						WithOperationID("testEndpoint"))
					return spec
				}(),
			},
			want: &openAPIDefinitions{
				Schemas: []*schemaDefinition{
					{
						Name: "TestRequest1",
						Fields: []schemaField{
							{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []schemaField{
							{Name: "result", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
				},
				Paths: []*pathDefinition{
					{
						Path: "/test",
						Operations: []pathOperation{
							{
								Method:      HTTPMethodPost,
								OperationID: "testEndpoint",
								RequestRef:  "TestRequest1",
								RequestSchema: &schemaDefinition{
									Name: "TestRequest1",
									Fields: []schemaField{
										{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
									},
								},
								Responses: []operationResponse{
									{
										StatusCode:  StatusOK,
										ResponseRef: "TestResponse1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple specs with one endpoint each",
			specs: []AbstractOpenAPIServiceSpec{
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest(spec, state, HTTPMethodPost, "/endpoint1",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
							return OpenAPISendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("endpoint1"))
					return spec
				}(),
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService2{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest(spec, state, HTTPMethodGet, "/endpoint2",
						func(ctx context.Context, event *TestRequest2, sm *TestService2) OpenAPIResponse[*TestResponse2] {
							return OpenAPISendTo(ctx, sm, &TestResponse2{})
						},
						WithOperationID("endpoint2"))
					return spec
				}(),
			},
			want: &openAPIDefinitions{
				Schemas: []*schemaDefinition{
					{
						Name: "TestRequest1",
						Fields: []schemaField{
							{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []schemaField{
							{Name: "result", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestRequest2",
						Fields: []schemaField{
							{Name: "info", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse2",
						Fields: []schemaField{
							{Name: "value", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
				},
				Paths: []*pathDefinition{
					{
						Path: "/endpoint1",
						Operations: []pathOperation{
							{
								Method:      HTTPMethodPost,
								OperationID: "endpoint1",
								RequestRef:  "TestRequest1",
								RequestSchema: &schemaDefinition{
									Name: "TestRequest1",
									Fields: []schemaField{
										{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
									},
								},
								Responses: []operationResponse{
									{
										StatusCode:  StatusOK,
										ResponseRef: "TestResponse1",
									},
								},
							},
						},
					},
					{
						Path: "/endpoint2",
						Operations: []pathOperation{
							{
								Method:      HTTPMethodGet,
								OperationID: "endpoint2",
								RequestRef:  "TestRequest2",
								RequestSchema: &schemaDefinition{
									Name: "TestRequest2",
									Fields: []schemaField{
										{Name: "info", Type: "string", Format: "", IsArray: false, Required: true},
									},
								},
								Responses: []operationResponse{
									{
										StatusCode:  StatusOK,
										ResponseRef: "TestResponse2",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "aggregates multiple responses for same operation",
			specs: []AbstractOpenAPIServiceSpec{
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest(spec, state, HTTPMethodGet, "/multi",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
							return OpenAPISendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("multiOp"),
						WithStatusCode(StatusOK))
					OnOpenAPIRequest(spec, state, HTTPMethodGet, "/multi",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
							return OpenAPISendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("multiOp"),
						WithStatusCode(StatusBadRequest))
					return spec
				}(),
			},
			want: &openAPIDefinitions{
				Schemas: []*schemaDefinition{
					{
						Name: "TestRequest1",
						Fields: []schemaField{
							{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []schemaField{
							{Name: "result", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
				},
				Paths: []*pathDefinition{
					{
						Path: "/multi",
						Operations: []pathOperation{
							{
								Method:      HTTPMethodGet,
								OperationID: "multiOp",
								RequestRef:  "TestRequest1",
								RequestSchema: &schemaDefinition{
									Name: "TestRequest1",
									Fields: []schemaField{
										{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
									},
								},
								Responses: []operationResponse{
									{StatusCode: StatusOK, ResponseRef: "TestResponse1"},
									{StatusCode: StatusBadRequest, ResponseRef: "TestResponse1"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newSchemaAnalyzer()
			got := a.analyzeSpecs(tt.specs...)

			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("analyzeSpecs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
