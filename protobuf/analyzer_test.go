package protobuf

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTypeAnalyzer_analyzeSpecs(t *testing.T) {
	tests := []struct {
		name    string
		specs   []AbstractProtobufServiceSpec
		want    *protoDefinitions
		wantErr bool
	}{
		{
			name:  "empty specs returns empty definitions",
			specs: []AbstractProtobufServiceSpec{},
			want: &protoDefinitions{
				Messages: []*protoMessage{},
				Services: []*protoService{},
			},
			wantErr: false,
		},
		{
			name: "single spec with one method",
			specs: []AbstractProtobufServiceSpec{
				func() AbstractProtobufServiceSpec {
					spec := NewProtobufServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnProtobufMessage(spec, state, "TestMethod",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) ProtobufResponse[*TestResponse1] {
							return ProtobufSendTo(ctx, sm, &TestResponse1{Result: "test"})
						})
					return spec
				}(),
			},
			want: &protoDefinitions{
				Messages: []*protoMessage{
					{
						Name: "TestRequest1",
						Fields: []protoField{
							{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []protoField{
							{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
						},
					},
				},
				Services: []*protoService{
					{
						Name: "TestService1",
						Methods: []protoMethod{
							{
								Name:       "TestMethod",
								InputType:  "TestRequest1",
								OutputType: "TestResponse1",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple specs with multiple methods",
			specs: []AbstractProtobufServiceSpec{
				func() AbstractProtobufServiceSpec {
					spec := NewProtobufServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnProtobufMessage(spec, state, "Method1",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) ProtobufResponse[*TestResponse1] {
							return ProtobufSendTo(ctx, sm, &TestResponse1{})
						})
					OnProtobufMessage(spec, state, "Method2",
						func(ctx context.Context, event *TestRequest2, sm *TestService1) ProtobufResponse[*TestResponse2] {
							return ProtobufSendTo(ctx, sm, &TestResponse2{})
						})
					return spec
				}(),
				func() AbstractProtobufServiceSpec {
					spec := NewProtobufServiceSpec(&TestService2{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnProtobufMessage(spec, state, "Method3",
						func(ctx context.Context, event *TestRequest3, sm *TestService2) ProtobufResponse[*TestResponse3] {
							return ProtobufSendTo(ctx, sm, &TestResponse3{})
						})
					return spec
				}(),
			},
			want: &protoDefinitions{
				Messages: []*protoMessage{
					{
						Name: "TestRequest1",
						Fields: []protoField{
							{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestRequest2",
						Fields: []protoField{
							{Name: "Info", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []protoField{
							{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse2",
						Fields: []protoField{
							{Name: "Value", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestRequest3",
						Fields: []protoField{
							{Name: "Input", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse3",
						Fields: []protoField{
							{Name: "Output", Type: "string", Number: 1, IsRepeated: false},
						},
					},
				},
				Services: []*protoService{
					{
						Name: "TestService1",
						Methods: []protoMethod{
							{
								Name:       "Method1",
								InputType:  "TestRequest1",
								OutputType: "TestResponse1",
							},
							{
								Name:       "Method2",
								InputType:  "TestRequest2",
								OutputType: "TestResponse2",
							},
						},
					},
					{
						Name: "TestService2",
						Methods: []protoMethod{
							{
								Name:       "Method3",
								InputType:  "TestRequest3",
								OutputType: "TestResponse3",
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
			a := newTypeAnalyzer()
			got := a.analyzeSpecs(tt.specs...)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("analyzeSpecs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
