package protobuf

import (
	"context"
	"reflect"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
)

func TestOnProtobufMessage(t *testing.T) {
	tests := []struct {
		name           string
		methodName     string
		expectedMethod rpcMethod
	}{
		{
			name:       "registers single method and verifies goat integration",
			methodName: "TestMethod",
			expectedMethod: rpcMethod{
				ServiceType: "TestService1",
				MethodName:  "TestMethod",
				InputType:   "TestRequest1",
				OutputType:  "TestResponse1",
			},
		},
		{
			name:       "registers method with custom name",
			methodName: "CustomMethod",
			expectedMethod: rpcMethod{
				ServiceType: "TestService1",
				MethodName:  "CustomMethod",
				InputType:   "TestRequest1",
				OutputType:  "TestResponse1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := NewProtobufServiceSpec(&TestService1{})
			state := &TestIdleState{}
			spec.DefineStates(state).SetInitialState(state)

			initialMethodCount := len(spec.GetRPCMethods())

			OnProtobufMessage(spec, state, tt.methodName,
				func(ctx context.Context, event *TestRequest1, sm *TestService1) ProtobufResponse[*TestResponse1] {
					return ProtobufSendTo(ctx, sm, &TestResponse1{Result: "test"})
				})

			methods := spec.GetRPCMethods()
			if len(methods) != initialMethodCount+1 {
				t.Fatalf("expected %d method(s), got %d", initialMethodCount+1, len(methods))
			}

			if diff := cmp.Diff(tt.expectedMethod, methods[len(methods)-1]); diff != "" {
				t.Errorf("method mismatch (-want +got):\n%s", diff)
			}

			if spec.StateMachineSpec == nil {
				t.Fatal("StateMachineSpec should not be nil after OnProtobufMessage call")
			}

			if len(spec.GetRPCMethods()) == 0 {
				t.Error("RPC methods should be registered after OnProtobufMessage call")
			}
		})
	}
}

func TestGetServiceTypeName(t *testing.T) {
	tests := []struct {
		name string
		spec *goat.StateMachineSpec[*TestService1]
		want string
	}{
		{
			name: "returns correct type name",
			spec: goat.NewStateMachineSpec(&TestService1{}),
			want: "TestService1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getServiceTypeName(tt.spec)
			if got != tt.want {
				t.Errorf("getServiceTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEventTypeName(t *testing.T) {
	tests := []struct {
		name  string
		event AbstractProtobufMessage
		want  string
	}{
		{
			name:  "returns request type name",
			event: &TestRequest1{},
			want:  "TestRequest1",
		},
		{
			name:  "returns response type name",
			event: &TestResponse1{},
			want:  "TestResponse1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEventTypeName(tt.event)
			if got != tt.want {
				t.Errorf("getEventTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "handles struct value",
			value: TestService1{},
			want:  "TestService1",
		},
		{
			name:  "handles pointer to struct",
			value: &TestService1{},
			want:  "TestService1",
		},
		{
			name:  "handles event type",
			value: &TestRequest1{},
			want:  "TestRequest1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTypeName(tt.value)
			if got != tt.want {
				t.Errorf("getTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeMessage(t *testing.T) {
	tests := []struct {
		name     string
		instance AbstractProtobufMessage
		want     *protoMessage
	}{
		{
			name:     "analyzes TestRequest1 correctly",
			instance: &TestRequest1{},
			want: &protoMessage{
				Name: "TestRequest1",
				Fields: []protoField{
					{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
				},
			},
		},
		{
			name:     "analyzes TestResponse1 correctly",
			instance: &TestResponse1{},
			want: &protoMessage{
				Name: "TestResponse1",
				Fields: []protoField{
					{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeMessage(tt.instance)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("analyzeMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMapGoFieldToProto(t *testing.T) {
	tests := []struct {
		name          string
		goType        reflect.Type
		wantProtoType string
		wantRepeated  bool
	}{
		{
			name:          "maps string to string",
			goType:        reflect.TypeOf(""),
			wantProtoType: "string",
			wantRepeated:  false,
		},
		{
			name:          "maps bool to bool",
			goType:        reflect.TypeOf(false),
			wantProtoType: "bool",
			wantRepeated:  false,
		},
		{
			name:          "maps int64 to int64",
			goType:        reflect.TypeOf(int64(0)),
			wantProtoType: "int64",
			wantRepeated:  false,
		},
		{
			name:          "maps []string to repeated string",
			goType:        reflect.TypeOf([]string{}),
			wantProtoType: "string",
			wantRepeated:  true,
		},
		{
			name:          "handles unsupported type",
			goType:        reflect.TypeOf(make(chan int)),
			wantProtoType: "",
			wantRepeated:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotRepeated := mapGoFieldToProto(tt.goType)
			if gotType != tt.wantProtoType {
				t.Errorf("mapGoFieldToProto() type = %v, want %v", gotType, tt.wantProtoType)
			}
			if gotRepeated != tt.wantRepeated {
				t.Errorf("mapGoFieldToProto() repeated = %v, want %v", gotRepeated, tt.wantRepeated)
			}
		})
	}
}
