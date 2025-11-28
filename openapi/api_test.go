package openapi

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOnOpenAPIRequest(t *testing.T) {
	tests := []struct {
		name             string
		method           HTTPMethod
		path             string
		operationID      string
		expectedEndpoint endpointMetadata
		setup            func(spec *OpenAPIServiceSpec[*TestService1], state *TestIdleState)
	}{
		{
			name:        "registers single endpoint and verifies goat integration",
			method:      HTTPMethodPost,
			path:        "/test",
			operationID: "testEndpoint",
			expectedEndpoint: endpointMetadata{
				Path:         "/test",
				Method:       HTTPMethodPost,
				OperationID:  "testEndpoint",
				RequestType:  "TestRequest1",
				ResponseType: "TestResponse1",
				StatusCode:   StatusOK,
			},
		},
		{
			name:        "registers GET endpoint",
			method:      HTTPMethodGet,
			path:        "/custom/{id}",
			operationID: "customEndpoint",
			expectedEndpoint: endpointMetadata{
				Path:         "/custom/{id}",
				Method:       HTTPMethodGet,
				OperationID:  "customEndpoint",
				RequestType:  "TestRequest1",
				ResponseType: "TestResponse1",
				StatusCode:   StatusOK,
			},
		},
		{
			name:        "registers additional method on existing path",
			method:      HTTPMethodDelete,
			path:        "/test",
			operationID: "deleteTestEndpoint",
			setup: func(spec *OpenAPIServiceSpec[*TestService1], state *TestIdleState) {
				OnOpenAPIRequest(spec, state, HTTPMethodGet, "/test",
					func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
						return OpenAPISendTo(ctx, sm, &TestResponse1{Result: "existing"})
					},
					WithOperationID("existingEndpoint"))
			},
			expectedEndpoint: endpointMetadata{
				Path:         "/test",
				Method:       HTTPMethodDelete,
				OperationID:  "deleteTestEndpoint",
				RequestType:  "TestRequest1",
				ResponseType: "TestResponse1",
				StatusCode:   StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := NewOpenAPIServiceSpec(&TestService1{})
			state := &TestIdleState{}
			spec.DefineStates(state).SetInitialState(state)

			if tt.setup != nil {
				tt.setup(spec, state)
			}

			OnOpenAPIRequest(spec, state, tt.method, tt.path,
				func(ctx context.Context, event *TestRequest1, sm *TestService1) OpenAPIResponse[*TestResponse1] {
					return OpenAPISendTo(ctx, sm, &TestResponse1{Result: "test"})
				},
				WithOperationID(tt.operationID))

			endpoints := spec.getEndpoints()

			if diff := cmp.Diff(tt.expectedEndpoint, endpoints[len(endpoints)-1]); diff != "" {
				t.Errorf("endpoint mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnalyzeSchema(t *testing.T) {
	tests := []struct {
		name     string
		instance AbstractOpenAPISchema
		want     *schemaDefinition
	}{
		{
			name:     "analyzes TestRequest1 correctly",
			instance: &TestRequest1{},
			want: &schemaDefinition{
				Name: "TestRequest1",
				Fields: []schemaField{
					{Name: "data", Type: "string", Format: "", IsArray: false, Required: true},
				},
			},
		},
		{
			name:     "analyzes TestResponse1 correctly",
			instance: &TestResponse1{},
			want: &schemaDefinition{
				Name: "TestResponse1",
				Fields: []schemaField{
					{Name: "result", Type: "string", Format: "", IsArray: false, Required: true},
				},
			},
		},
		{
			name: "applies explicit parameter name when provided",
			instance: &struct {
				OpenAPISchema[*TestService1, *TestService1]
				UserID string `openapi:"path=user_id"`
			}{},
			want: &schemaDefinition{
				Name: "",
				Fields: []schemaField{
					{Name: "user_id", Type: "string", Format: "", IsArray: false, Required: true, ParamType: parameterTypePath},
				},
			},
		},
		{
			name: "falls back to snake case for unnamed parameters",
			instance: &struct {
				OpenAPISchema[*TestService1, *TestService1]
				SessionToken string `openapi:"query"`
			}{},
			want: &schemaDefinition{
				Name: "",
				Fields: []schemaField{
					{Name: "session_token", Type: "string", Format: "", IsArray: false, Required: false, ParamType: parameterTypeQuery},
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

func TestParseField(t *testing.T) {
	noTagField := structField(t, struct {
		Field string
	}{}, "Field")

	requiredOnlyField := structField(t, struct {
		Value string `openapi:"required"`
	}{}, "Value")

	pathWithExplicitNameField := structField(t, struct {
		ID string `openapi:" path = user_id "`
	}{}, "ID")

	pathWithoutExplicitNameField := structField(t, struct {
		SessionToken string `openapi:"path"`
	}{}, "SessionToken")

	queryWithRequiredField := structField(t, struct {
		Filter string `openapi:" query = search_term , required "`
	}{}, "Filter")

	requiredBeforeDefinitionField := structField(t, struct {
		Filter string `openapi:"required,query=search_term"`
	}{}, "Filter")

	headerImplicitNameField := structField(t, struct {
		UserID string `openapi:"header"`
	}{}, "UserID")

	unknownTypeWithNameField := structField(t, struct {
		Cookie string `openapi:"cookie=my_cookie"`
	}{}, "Cookie")

	unknownTypeRequiredField := structField(t, struct {
		SessionCookie string `openapi:"cookie,required"`
	}{}, "SessionCookie")

	missingParamNameField := structField(t, struct {
		Value string `openapi:"query="`
	}{}, "Value")

	missingParamTypeField := structField(t, struct {
		Value string `openapi:"=user_id"`
	}{}, "Value")

	tests := []struct {
		name          string
		field         *reflect.StructField
		wantFieldName string
		wantType      parameterType
		wantRequired  bool
	}{
		{
			name:          "no tag returns defaults",
			field:         noTagField,
			wantFieldName: "field",
			wantType:      parameterTypeNone,
			wantRequired:  false,
		},
		{
			name:          "required only tag marks field required",
			field:         requiredOnlyField,
			wantFieldName: "value",
			wantType:      parameterTypeNone,
			wantRequired:  true,
		},
		{
			name:          "path parameter with explicit name is always required",
			field:         pathWithExplicitNameField,
			wantFieldName: "user_id",
			wantType:      parameterTypePath,
			wantRequired:  true,
		},
		{
			name:          "path parameter without explicit name uses snake case",
			field:         pathWithoutExplicitNameField,
			wantFieldName: "session_token",
			wantType:      parameterTypePath,
			wantRequired:  true,
		},
		{
			name:          "query parameter trims spaces and applies required flag",
			field:         queryWithRequiredField,
			wantFieldName: "search_term",
			wantType:      parameterTypeQuery,
			wantRequired:  true,
		},
		{
			name:          "required modifier before definition still applies",
			field:         requiredBeforeDefinitionField,
			wantFieldName: "search_term",
			wantType:      parameterTypeQuery,
			wantRequired:  true,
		},
		{
			name:          "header parameter without explicit name uses snake case",
			field:         headerImplicitNameField,
			wantFieldName: "user_id",
			wantType:      parameterTypeHeader,
			wantRequired:  false,
		},
		{
			name:          "unknown parameter type with explicit name falls back to defaults",
			field:         unknownTypeWithNameField,
			wantFieldName: "cookie",
			wantType:      parameterTypeNone,
			wantRequired:  false,
		},
		{
			name:          "unknown parameter type still honors required flag",
			field:         unknownTypeRequiredField,
			wantFieldName: "sessionCookie",
			wantType:      parameterTypeNone,
			wantRequired:  true,
		},
		{
			name:          "missing parameter name falls back to defaults",
			field:         missingParamNameField,
			wantFieldName: "value",
			wantType:      parameterTypeNone,
			wantRequired:  false,
		},
		{
			name:          "missing parameter type falls back to defaults",
			field:         missingParamTypeField,
			wantFieldName: "value",
			wantType:      parameterTypeNone,
			wantRequired:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFieldName, gotType, gotRequired := parseField(tt.field)
			if gotFieldName != tt.wantFieldName {
				t.Errorf("parseField() fieldName = %v, want %v", gotFieldName, tt.wantFieldName)
			}
			if gotType != tt.wantType {
				t.Errorf("parseField() type = %v, want %v", gotType, tt.wantType)
			}
			if gotRequired != tt.wantRequired {
				t.Errorf("parseField() required = %v, want %v", gotRequired, tt.wantRequired)
			}
		})
	}
}

func structField(t *testing.T, sample any, fieldName string) *reflect.StructField {
	t.Helper()

	typ := reflect.TypeOf(sample)
	if typ.Kind() != reflect.Struct {
		t.Fatalf("structField sample must be a struct, got %T", sample)
	}

	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("field %s not found in %#v", fieldName, sample)
	}

	fieldCopy := field
	return &fieldCopy
}
