package goat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzePackageWithTypes(t *testing.T) {
	// Test with the client-server example
	packagePath := "./example/client-server"
	outputPath := filepath.Join(t.TempDir(), "diagram.md")

	err := AnalyzePackage(packagePath, outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	// Check if file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify basic structure
	contentStr := string(content)
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
	outputPath := filepath.Join(t.TempDir(), "diagram.md")

	err := AnalyzePackage(packagePath, outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	// Check if file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	
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
		},
		{
			From:             "Server",
			To:               "Client",
			EventType:        "eCheckMenuExistenceResponse",
			HandlerType:      "OnEvent",
			HandlerEventType: "eCheckMenuExistenceRequest",
		},
	}

	result := generateMermaid(flows)

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

func TestBuildRequestResponsePairs(t *testing.T) {
	flows := []CommunicationFlow{
		// OnEntry initiates request
		{
			From:             "Client",
			To:               "Server",
			EventType:        "Request1",
			HandlerType:      "OnEntry",
			HandlerEventType: "",
		},
		// OnEvent handles and responds
		{
			From:             "Server",
			To:               "Client",
			EventType:        "Response1",
			HandlerType:      "OnEvent",
			HandlerEventType: "Request1",
		},
		// Another OnEvent (should not be paired)
		{
			From:             "Server",
			To:               "Client",
			EventType:        "Response2",
			HandlerType:      "OnEvent",
			HandlerEventType: "Request2",
		},
	}

	paired := buildRequestResponsePairs(flows)

	// Should have all 3 flows, with the paired ones first
	if len(paired) != 3 {
		t.Errorf("Expected 3 flows, got %d", len(paired))
	}

	// First should be the OnEntry request
	if paired[0].HandlerType != "OnEntry" {
		t.Error("First flow should be OnEntry")
	}

	// Second should be the corresponding OnEvent response
	if len(paired) > 1 && (paired[1].HandlerType != "OnEvent" || paired[1].HandlerEventType != "Request1") {
		t.Error("Second flow should be OnEvent response to Request1")
	}
}

func TestAnalyzePackage_ClientServer_Golden(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "sequence.md")

	err := AnalyzePackage("example/client-server", outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	want, err := os.ReadFile("example/client-server/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnalyzePackage_MeetingRoomWithoutExclusion_Golden(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "sequence.md")

	err := AnalyzePackage("example/meeting-room-reservation/without-exclusion", outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	want, err := os.ReadFile("example/meeting-room-reservation/without-exclusion/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnalyzePackage_MeetingRoomWithExclusion_Golden(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "sequence.md")

	err := AnalyzePackage("example/meeting-room-reservation/with-exclusion", outputPath)
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	want, err := os.ReadFile("example/meeting-room-reservation/with-exclusion/sequence.md")
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("Generated sequence diagram doesn't match expected:\ngot:\n%s\nwant:\n%s", got, want)
	}
}