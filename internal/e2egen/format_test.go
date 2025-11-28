package e2egen

import (
	"testing"
)

func TestFormatStructLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkgAlias string
		typeName string
		data     map[string]any
		want     string
	}{
		{
			name:     "empty data",
			pkgAlias: "pb",
			typeName: "Request",
			data:     map[string]any{},
			want:     "&pb.Request{}",
		},
		{
			name:     "populated data",
			pkgAlias: "pb",
			typeName: "Response",
			data: map[string]any{
				"Count":  5,
				"Name":   "alice",
				"Active": true,
			},
			want: "&pb.Response{\n\t\t\t\tActive: true,\n\t\t\t\tCount: 5,\n\t\t\t\tName: \"alice\",\n\t\t\t}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatStructLiteral(tt.pkgAlias, tt.typeName, tt.data); got != tt.want {
				t.Fatalf("FormatStructLiteral() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	var nilPtr *int

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "string", value: "hello", want: "\"hello\""},
		{name: "bool", value: true, want: "true"},
		{name: "int", value: int64(42), want: "42"},
		{name: "uint", value: uint(7), want: "7"},
		{name: "float", value: 3.5, want: "3.5"},
		{name: "slice", value: []int{1, 2}, want: "[]int{1, 2}"},
		{name: "nil interface", value: nil, want: "nil"},
		{name: "nil pointer", value: nilPtr, want: "nil"},
		{name: "nil slice", value: []string(nil), want: "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatValue(tt.value); got != tt.want {
				t.Fatalf("FormatValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
