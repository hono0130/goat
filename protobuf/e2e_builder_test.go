package protobuf

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
)

func TestSerializeMessage(t *testing.T) {
	type embeddedFields struct {
		ShouldBeIgnored string
	}

	type sampleMessage struct {
		Message[*TestService1, *TestService1]

		Foo    string
		UserID string
		Count  int
		Event  goat.Event[*TestService1, *TestService1]
		hidden string
		_      int
		embeddedFields
	}

	msg := &sampleMessage{
		Foo:    "foo",
		UserID: "user-1",
		Count:  3,
		Event:  goat.Event[*TestService1, *TestService1]{},
		hidden: "secret",
		embeddedFields: embeddedFields{
			ShouldBeIgnored: "ignore",
		},
	}

	t.Run("filters non-protobuf fields and converts field names", func(t *testing.T) {
		got := serializeMessage(msg)
		want := map[string]any{
			"Foo":    "foo",
			"UserId": "user-1",
			"Count":  3,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("serializeMessage() mismatch (-want +got):\n%s", diff)
		}
	})
}
