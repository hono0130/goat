package openapi

import (
	"context"
	"reflect"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
)

func TestOnOpenAPIEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		operationID      string
		expectedEndpoint endpointMetadata
	}{
		{
			name:        "registers single endpoint and verifies goat integration",
			method:      "POST",
			path:        "/test",
			operationID: "testEndpoint",
			expectedEndpoint: endpointMetadata{
				Path:         "/test",
				Method:       "POST",
				OperationID:  "testEndpoint",
				RequestType:  "TestRequest1",
				ResponseType: "TestResponse1",
			},
		},
		{
			name:        "registers GET endpoint",
			method:      "GET",
			path:        "/custom/{id}",
			operationID: "customEndpoint",
			expectedEndpoint: endpointMetadata{
				Path:         "/custom/{id}",
				Method:       "GET",
				OperationID:  "customEndpoint",
				RequestType:  "TestRequest1",
				ResponseType: "TestResponse1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := NewOpenAPIServiceSpec(&TestService1{})
			state := &TestIdleState{}
			spec.DefineStates(state).SetInitialState(state)

			initialEndpointCount := len(spec.GetEndpoints())

			OnOpenAPIEndpoint[*TestService1, *TestRequest1, *TestResponse1](spec, state, tt.method, tt.path, tt.operationID,
				func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
					return OpenAPISendTo(ctx, sm, &TestResponse1{Result: "test"})
				})

			endpoints := spec.GetEndpoints()
			if len(endpoints) != initialEndpointCount+1 {
				t.Fatalf("expected %d endpoint(s), got %d", initialEndpointCount+1, len(endpoints))
			}

			if diff := cmp.Diff(tt.expectedEndpoint, endpoints[len(endpoints)-1]); diff != "" {
				t.Errorf("endpoint mismatch (-want +got):\n%s", diff)
			}

			if spec.StateMachineSpec == nil {
				t.Fatal("StateMachineSpec should not be nil after OnOpenAPIEndpoint call")
			}

			if len(spec.GetEndpoints()) == 0 {
				t.Error("Endpoints should be registered after OnOpenAPIEndpoint call")
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
		event AbstractOpenAPIEndpoint
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

func TestAnalyzeSchema(t *testing.T) {
	tests := []struct {
		name     string
		instance AbstractOpenAPIEndpoint
		want     *schemaDefinition
	}{
		{
			name:     "analyzes TestRequest1 correctly",
			instance: &TestRequest1{},
			want: &schemaDefinition{
				Name: "TestRequest1",
				Fields: []schemaField{
					{Name: "Data", Type: "string", Format: "", IsArray: false, Required: true},
				},
			},
		},
		{
			name:     "analyzes TestResponse1 correctly",
			instance: &TestResponse1{},
			want: &schemaDefinition{
				Name: "TestResponse1",
				Fields: []schemaField{
					{Name: "Result", Type: "string", Format: "", IsArray: false, Required: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeSchema(tt.instance)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("analyzeSchema() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMapGoFieldToOpenAPI(t *testing.T) {
	tests := []struct {
		name        string
		goType      reflect.Type
		wantType    string
		wantFormat  string
		wantIsArray bool
	}{
		{
			name:        "maps string to string",
			goType:      reflect.TypeOf(""),
			wantType:    "string",
			wantFormat:  "",
			wantIsArray: false,
		},
		{
			name:        "maps bool to boolean",
			goType:      reflect.TypeOf(false),
			wantType:    "boolean",
			wantFormat:  "",
			wantIsArray: false,
		},
		{
			name:        "maps int64 to integer with int64 format",
			goType:      reflect.TypeOf(int64(0)),
			wantType:    "integer",
			wantFormat:  "int64",
			wantIsArray: false,
		},
		{
			name:        "maps int32 to integer with int32 format",
			goType:      reflect.TypeOf(int32(0)),
			wantType:    "integer",
			wantFormat:  "int32",
			wantIsArray: false,
		},
		{
			name:        "maps float64 to number with double format",
			goType:      reflect.TypeOf(float64(0)),
			wantType:    "number",
			wantFormat:  "double",
			wantIsArray: false,
		},
		{
			name:        "maps []string to array of strings",
			goType:      reflect.TypeOf([]string{}),
			wantType:    "string",
			wantFormat:  "",
			wantIsArray: true,
		},
		{
			name:        "handles unsupported type",
			goType:      reflect.TypeOf(make(chan int)),
			wantType:    "",
			wantFormat:  "",
			wantIsArray: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotFormat, gotIsArray := mapGoFieldToOpenAPI(tt.goType)
			if gotType != tt.wantType {
				t.Errorf("mapGoFieldToOpenAPI() type = %v, want %v", gotType, tt.wantType)
			}
			if gotFormat != tt.wantFormat {
				t.Errorf("mapGoFieldToOpenAPI() format = %v, want %v", gotFormat, tt.wantFormat)
			}
			if gotIsArray != tt.wantIsArray {
				t.Errorf("mapGoFieldToOpenAPI() isArray = %v, want %v", gotIsArray, tt.wantIsArray)
			}
		})
	}
}
