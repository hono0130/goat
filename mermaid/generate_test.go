package mermaid

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerate(t *testing.T) {
	t.Setenv("GOCACHE", t.TempDir())

	var buf bytes.Buffer
	dir := writeWorkflowFixture(t)
	if err := Generate(dir, &buf); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	const want = "sequenceDiagram\n" +
		"    participant Sender\n" +
		"    participant Receiver\n" +
		"    participant Logger\n" +
		"\n" +
		"    Sender->>Receiver: PingEvent\n" +
		"    alt\n" +
		"        Receiver->>Logger: NotifyEvent\n" +
		"    else\n" +
		"        Receiver->>Sender: AckEvent\n" +
		"        Sender->>Receiver: NotifyEvent\n" +
		"    end\n"

	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Fatalf("Generate output mismatch (-want +got):\n%s", diff)
	}
}

func TestOrderedParticipants(t *testing.T) {
	elements := []element{
		{flows: []flow{
			{from: "Receiver", to: "Logger"},
			{from: "ExtraB", to: "Sender"},
		}},
		{branches: [][]flow{{
			{from: "Alpha", to: "Beta"},
		}}},
	}
	order := []string{"Sender", "Receiver"}
	got := orderedParticipants(elements, order)
	want := []string{"Sender", "Receiver", "Alpha", "Beta", "ExtraB", "Logger"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("orderedParticipants mismatch (-want +got):\n%s", diff)
	}
}
