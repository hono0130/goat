package goat

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestAnalyzePackage(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		golden string
	}{
		{
			name:   "ClientServer",
			path:   "example/client-server",
			golden: "example/client-server/sequence.md",
		},
		{
			name:   "MeetingRoomWithoutExclusion",
			path:   "example/meeting-room-reservation/without-exclusion",
			golden: "example/meeting-room-reservation/without-exclusion/sequence.md",
		},
		{
			name:   "MeetingRoomWithExclusion",
			path:   "example/meeting-room-reservation/with-exclusion",
			golden: "example/meeting-room-reservation/with-exclusion/sequence.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := AnalyzePackage(tt.path, &buf); err != nil {
				t.Fatalf("AnalyzePackage failed: %v", err)
			}
			got := buf.String()
			want, err := os.ReadFile(tt.golden)
			if err != nil {
				t.Fatalf("Failed to read expected file: %v", err)
			}
			if got != string(want) {
				t.Errorf("generated sequence diagram doesn't match expected:\n got:\n%s\n want:\n%s", got, want)
			}
		})
	}
}

func TestGenerateMermaid(t *testing.T) {
	flows := []CommunicationFlow{
		{
			From:             "Client",
			To:               "Server",
			EventType:        "eCheckMenuExistenceRequest",
			HandlerType:      "OnEntry",
			HandlerEventType: "",
			HandlerID:        "Client_OnEntry_",
		},
		{
			From:             "Server",
			To:               "Client",
			EventType:        "eCheckMenuExistenceResponse",
			HandlerType:      "OnEvent",
			HandlerEventType: "eCheckMenuExistenceRequest",
			HandlerID:        "Server_OnEvent_eCheckMenuExistenceRequest",
		},
	}

	elements := buildSequenceDiagramElements(flows)
	result := generateMermaid(elements, []string{"Client", "Server"})

	if !strings.Contains(result, "sequenceDiagram") {
		t.Error("Should contain sequenceDiagram header")
	}
	if !strings.Contains(result, "participant Client") {
		t.Error("Should declare Client participant")
	}
	if !strings.Contains(result, "participant Server") {
		t.Error("Should declare Server participant")
	}
	if !strings.Contains(result, "Client->>Server: eCheckMenuExistenceRequest") {
		t.Error("Should contain Client to Server request")
	}
	if !strings.Contains(result, "Server->>Client: eCheckMenuExistenceResponse") {
		t.Error("Should contain Server to Client response")
	}
}

func TestBuildSequenceDiagramElements(t *testing.T) {
	flows := []CommunicationFlow{
		{
			From:             "Client",
			To:               "Server",
			EventType:        "Request1",
			HandlerType:      "OnEntry",
			HandlerEventType: "",
			HandlerID:        "Client_OnEntry_",
		},
		{
			From:             "Server",
			To:               "Client",
			EventType:        "Response1",
			HandlerType:      "OnEvent",
			HandlerEventType: "Request1",
			HandlerID:        "Server_OnEvent_Request1",
		},
	}

	elements := buildSequenceDiagramElements(flows)
	if len(elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elements))
	}
	if elements[0].IsOptional {
		t.Error("first element should not be optional")
	}
	if elements[1].IsOptional {
		t.Error("second element should not be optional")
	}
}
