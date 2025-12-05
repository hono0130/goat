package protobuf

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTypeAnalyzer_analyzeSpecs(t *testing.T) {
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
				Messages: []*message{},
				Services: []*service{},
			},
			wantErr: false,
		},
		{
			name: "single spec with one method",
			specs: []AbstractServiceSpec{
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnMessage(spec, state, "TestMethod",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{Result: "test"})
						})
					return spec
				}(),
			},
			want: &definitions{
				Messages: []*message{
					{
						Name: "TestRequest1",
						Fields: []field{
							{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []field{
							{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
						},
					},
				},
				Services: []*service{
					{
						Name: "TestService1",
						Methods: []method{
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
			name: "multiple specs with one method each",
			specs: []AbstractServiceSpec{
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService1{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnMessage(spec, state, "Method1",
						func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
							return SendTo(ctx, sm, &TestResponse1{})
						})
					return spec
				}(),
				func() AbstractServiceSpec {
					spec := NewServiceSpec(&TestService2{})
					state := &TestIdleState{}
					spec.DefineStates(state).SetInitialState(state)
					OnMessage(spec, state, "Method2",
						func(ctx context.Context, event *TestRequest2, sm *TestService2) Response[*TestResponse2] {
							return SendTo(ctx, sm, &TestResponse2{})
						})
					return spec
				}(),
			},
			want: &definitions{
				Messages: []*message{
					{
						Name: "TestRequest1",
						Fields: []field{
							{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse1",
						Fields: []field{
							{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestRequest2",
						Fields: []field{
							{Name: "Info", Type: "string", Number: 1, IsRepeated: false},
						},
					},
					{
						Name: "TestResponse2",
						Fields: []field{
							{Name: "Value", Type: "string", Number: 1, IsRepeated: false},
						},
					},
				},
				Services: []*service{
					{
						Name: "TestService1",
						Methods: []method{
							{
								Name:       "Method1",
								InputType:  "TestRequest1",
								OutputType: "TestResponse1",
							},
						},
					},
					{
						Name: "TestService2",
						Methods: []method{
							{
								Name:       "Method2",
								InputType:  "TestRequest2",
								OutputType: "TestResponse2",
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
