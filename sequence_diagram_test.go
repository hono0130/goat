package goat

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestAnalyzePackageWithTypes(t *testing.T) {
	// Test with the client-server example
	packagePath := "./example/client-server"
	var buf bytes.Buffer

	err := AnalyzePackage(packagePath, &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	// Verify basic structure
	contentStr := buf.String()
	if !strings.Contains(contentStr, "sequenceDiagram") {
		t.Error("Output should contain 'sequenceDiagram'")
	}

	if !strings.Contains(contentStr, "participant Client") {
		t.Error("Output should contain 'participant Client'")
	}

	if !strings.Contains(contentStr, "participant Server") {
		t.Error("Output should contain 'participant Server'")
	}

	// Check for communication flows (should NOT contain From anymore)
	if strings.Contains(contentStr, "participant From") {
		t.Error("Output should NOT contain 'participant From' - this should be resolved to Client")
	}

	// Check for proper Client to Server to Client flow
	if !strings.Contains(contentStr, "Client->>Server:") {
		t.Error("Output should contain Client to Server communication")
	}

	if !strings.Contains(contentStr, "Server->>Client:") {
		t.Error("Output should contain Server to Client communication")
	}
}

func TestAnalyzePackageMeetingRoom(t *testing.T) {
	// Test with the meeting-room-reservation example
	packagePath := "./example/meeting-room-reservation/with-exclusion"
	var buf bytes.Buffer

	err := AnalyzePackage(packagePath, &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	contentStr := buf.String()

	// Should contain all three state machines
	expectedParticipants := []string{"ClientStateMachine", "ServerStateMachine", "DBStateMachine"}
	for _, participant := range expectedParticipants {
		if !strings.Contains(contentStr, "participant "+participant) {
			t.Errorf("Output should contain 'participant %s'", participant)
		}
	}

	// Should contain proper communications
	if !strings.Contains(contentStr, "ClientStateMachine->>ServerStateMachine:") {
		t.Error("Output should contain ClientStateMachine to ServerStateMachine communication")
	}

	if !strings.Contains(contentStr, "ServerStateMachine->>DBStateMachine:") {
		t.Error("Output should contain ServerStateMachine to DBStateMachine communication")
	}
}

func TestExtractEventTypeFromExpr(t *testing.T) {
	// This test requires actual AST nodes, so we'll test the fallback function
	tests := []struct {
		input    string
		expected string
	}{
		// These would be tested with actual type information in integration tests
	}

	for _, test := range tests {
		// Note: This would require creating actual AST nodes
		// For now, we rely on integration tests
		_ = test
	}
}

func TestResolveTargetType(t *testing.T) {
	// This test requires actual type information from go/packages
	// It's best tested through integration tests with real packages
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
	result := generateMermaidWithGroups(elements, []string{"Client", "Server"})

	// Check structure
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
		// OnEntry initiates request
		{
			From:             "Client",
			To:               "Server",
			EventType:        "Request1",
			HandlerType:      "OnEntry",
			HandlerEventType: "",
			HandlerID:        "Client_OnEntry_",
		},
		// OnEvent handles and responds
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

	// Should have 2 single elements (no conditional branches)
	if len(elements) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(elements))
	}

	// First should be the OnEntry request
	if elements[0].IsOptional {
		t.Error("First element should not be optional")
	}

	// Second should be the OnEvent response
	if len(elements) > 1 && elements[1].IsOptional {
		t.Error("Second element should not be optional")
	}
}

func TestAnalyzePackage_ClientServer_Golden(t *testing.T) {
	var buf bytes.Buffer

	err := AnalyzePackage("example/client-server", &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got := buf.String()

	want, err := os.ReadFile("example/client-server/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if got != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnalyzePackage_MeetingRoomWithoutExclusion_Golden(t *testing.T) {
	var buf bytes.Buffer

	err := AnalyzePackage("example/meeting-room-reservation/without-exclusion", &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got := buf.String()

	want, err := os.ReadFile("example/meeting-room-reservation/without-exclusion/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if got != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnalyzePackage_MeetingRoomWithExclusion_Golden(t *testing.T) {
	var buf bytes.Buffer

	err := AnalyzePackage("example/meeting-room-reservation/with-exclusion", &buf)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got := buf.String()

	want, err := os.ReadFile("example/meeting-room-reservation/with-exclusion/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if got != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
