package typeutil

import (
	"testing"
)

type sampleStruct struct{}

func TestName(t *testing.T) {
	t.Parallel()

	var value sampleStruct
	var nilPointer *sampleStruct

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "Nil", input: nil, want: ""},
		{name: "StructValue", input: value, want: "sampleStruct"},
		{name: "PointerValue", input: &value, want: "sampleStruct"},
		{name: "NilPointer", input: nilPointer, want: "sampleStruct"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Name(tt.input); got != tt.want {
				t.Fatalf("Name(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
