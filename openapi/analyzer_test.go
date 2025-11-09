package openapi

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
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
					OnOpenAPIRequest[*TestService1, *TestRequest1, *TestResponse1](spec, state, "POST", "/test",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) *TestResponse1 {
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
							{Name: "Data", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []schemaField{
							{Name: "Result", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
				},
				Paths: []*pathDefinition{
					{
						Path: "/test",
						Operations: []pathOperation{
							{
								Method:      "POST",
								OperationID: "testEndpoint",
								RequestRef:  "TestRequest1",
								ResponseRef: "TestResponse1",
								StatusCode:  StatusOK,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple specs with multiple endpoints",
			specs: []AbstractOpenAPIServiceSpec{
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest[*TestService1, *TestRequest1, *TestResponse1](spec, state, "POST", "/endpoint1",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) *TestResponse1 {
							return OpenAPISendTo(ctx, sm, &TestResponse1{})
						},
						WithOperationID("endpoint1"))
					OnOpenAPIRequest[*TestService1, *TestRequest2, *TestResponse2](spec, state, "GET", "/endpoint2",
						func(ctx context.Context, event *TestRequest2, sm *TestService1) *TestResponse2 {
							return OpenAPISendTo(ctx, sm, &TestResponse2{})
						},
						WithOperationID("endpoint2"))
					return spec
				}(),
				func() AbstractOpenAPIServiceSpec {
					spec := NewOpenAPIServiceSpec(&TestService2{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnOpenAPIRequest[*TestService2, *TestRequest3, *TestResponse3](spec, state, "PUT", "/endpoint3",
						func(ctx context.Context, event *TestRequest3, sm *TestService2) *TestResponse3 {
							return OpenAPISendTo(ctx, sm, &TestResponse3{})
						},
						WithOperationID("endpoint3"))
					return spec
				}(),
			},
			want: &openAPIDefinitions{
				Schemas: []*schemaDefinition{
					{
						Name: "TestRequest1",
						Fields: []schemaField{
							{Name: "Data", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestRequest2",
						Fields: []schemaField{
							{Name: "Info", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []schemaField{
							{Name: "Result", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse2",
						Fields: []schemaField{
							{Name: "Value", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestRequest3",
						Fields: []schemaField{
							{Name: "Input", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
					{
						Name: "TestResponse3",
						Fields: []schemaField{
							{Name: "Output", Type: "string", Format: "", IsArray: false, Required: true},
						},
					},
				},
				Paths: []*pathDefinition{
					{
						Path: "/endpoint1",
						Operations: []pathOperation{
							{
								Method:      "POST",
								OperationID: "endpoint1",
								RequestRef:  "TestRequest1",
								ResponseRef: "TestResponse1",
								StatusCode:  StatusOK,
							},
						},
					},
					{
						Path: "/endpoint2",
						Operations: []pathOperation{
							{
								Method:      "GET",
								OperationID: "endpoint2",
								RequestRef:  "TestRequest2",
								ResponseRef: "TestResponse2",
								StatusCode:  StatusOK,
							},
						},
					},
					{
						Path: "/endpoint3",
						Operations: []pathOperation{
							{
								Method:      "PUT",
								OperationID: "endpoint3",
								RequestRef:  "TestRequest3",
								ResponseRef: "TestResponse3",
								StatusCode:  StatusOK,
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

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("analyzeSpecs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
