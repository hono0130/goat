package mermaid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestStateMachineOrder(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	got := stateMachineOrder(pkg)
	want := []string{"Sender", "Receiver", "Logger"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("stateMachineOrder mismatch (-want +got):\n%s", diff)
	}
}

func TestCommunicationFlows(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	got := communicationFlows(pkg)
	want := []Flow{
		{From: "Sender", To: "Receiver", EventType: "PingEvent", HandlerType: "OnEntry"},
		{From: "Receiver", To: "Sender", EventType: "AckEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
		{From: "Receiver", To: "Logger", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
		{From: "Sender", To: "Receiver", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "AckEvent"},
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Flow{}, "HandlerID")); diff != "" {
		t.Fatalf("communicationFlows mismatch (-want +got):\n%s", diff)
	}
	for i, f := range got {
		if f.HandlerID == "" {
			t.Fatalf("flow[%d] handlerID is empty", i)
		}
	}
}

func TestBuildElements(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	flows := communicationFlows(pkg)
	got := buildElements(flows)
	want := []Element{
		{
			Flows: []Flow{{From: "Sender", To: "Receiver", EventType: "PingEvent", HandlerType: "OnEntry"}},
		},
		{
			Flows: nil,
			Branches: [][]Flow{
				{{From: "Receiver", To: "Logger", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"}},
				{
					{From: "Receiver", To: "Sender", EventType: "AckEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
					{From: "Sender", To: "Receiver", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "AckEvent"},
				},
			},
		},
	}
	if diff := cmp.Diff(
		want,
		got,
		cmpopts.IgnoreFields(Flow{}, "HandlerID"),
	); diff != "" {
		t.Fatalf("buildElements mismatch (-want +got):\n%s", diff)
	}
}

func TestAnalyze(t *testing.T) {
	dir := writeWorkflowFixture(t)
	diagram, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if diff := cmp.Diff([]string{"Sender", "Receiver", "Logger"}, diagram.Participants); diff != "" {
		t.Fatalf("participants mismatch (-want +got):\n%s", diff)
	}
	if len(diagram.Elements) == 0 {
		t.Fatal("expected elements to be populated")
	}
}
