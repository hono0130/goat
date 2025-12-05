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
		specs   []AbstractServiceSpec
		want    *definitions
		wantErr bool
	}{
		{
			name:  "empty specs returns empty definitions",
			specs: []AbstractServiceSpec{},
			want: &definitions{
				Schemas: []*schemaDefinition{},
				Paths:   []*pathDefinition{},
			},
			wantErr: false,
		},
		{
			name: "single spec with one endpoint",
			specs: []AbstractServiceSpec{
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnRequest(spec, state, HTTPMethodPost, "/test",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{Result: "test"})
						},
						WithOperationID("testEndpoint"))
					return spec
				}(),
			},
			want: &definitions{
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
			specs: []AbstractServiceSpec{
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnRequest(spec, state, HTTPMethodPost, "/endpoint1",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("endpoint1"))
					return spec
				}(),
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService2{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnRequest(spec, state, HTTPMethodGet, "/endpoint2",
						func(ctx context.Context, event *TestRequest2, sm *TestService2) Response[*TestResponse2] {
							return SendTo(ctx, sm, &TestResponse2{})
						},
						WithOperationID("endpoint2"))
					return spec
				}(),
			},
			want: &definitions{
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
			specs: []AbstractServiceSpec{
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnRequest(spec, state, HTTPMethodGet, "/multi",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("multiOp"),
						WithStatusCode(StatusOK))
					OnRequest(spec, state, HTTPMethodGet, "/multi",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("multiOp"),
						WithStatusCode(StatusBadRequest))
					return spec
				}(),
			},
			want: &definitions{
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
