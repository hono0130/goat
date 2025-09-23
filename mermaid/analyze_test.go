package mermaid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type simpleFlow struct {
	From             string
	To               string
	EventType        string
	HandlerType      string
	HandlerEventType string
}

func simplifyFlows(flows []flow) []simpleFlow {
	simplified := make([]simpleFlow, len(flows))
	for i, f := range flows {
		simplified[i] = simpleFlow{
			From:             f.from,
			To:               f.to,
			EventType:        f.eventType,
			HandlerType:      f.handlerType,
			HandlerEventType: f.handlerEventType,
		}
	}
	return simplified
}

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
	flows := communicationFlows(pkg)
	got := simplifyFlows(flows)
	want := []simpleFlow{
		{From: "Sender", To: "Receiver", EventType: "PingEvent", HandlerType: "OnEntry"},
		{From: "Receiver", To: "Sender", EventType: "AckEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
		{From: "Receiver", To: "Logger", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
		{From: "Sender", To: "Receiver", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "AckEvent"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("communicationFlows mismatch (-want +got):\n%s", diff)
	}
	for i, f := range flows {
		if f.handlerID == "" {
			t.Fatalf("flow[%d] handlerID is empty", i)
		}
	}
}

type simpleElement struct {
	Flows    []simpleFlow
	Branches [][]simpleFlow
}

func simplifyElements(elements []element) []simpleElement {
	simplified := make([]simpleElement, len(elements))
	for i, e := range elements {
		simplified[i].Flows = simplifyFlows(e.flows)
		if len(e.branches) == 0 {
			continue
		}
		simplified[i].Branches = make([][]simpleFlow, len(e.branches))
		for j, br := range e.branches {
			simplified[i].Branches[j] = simplifyFlows(br)
		}
	}
	return simplified
}

func TestBuildElements(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	flows := communicationFlows(pkg)
	elements := buildElements(flows)
	got := simplifyElements(elements)
	want := []simpleElement{
		{
			Flows: []simpleFlow{{From: "Sender", To: "Receiver", EventType: "PingEvent", HandlerType: "OnEntry"}},
		},
		{
			Flows: []simpleFlow{},
			Branches: [][]simpleFlow{
				{{From: "Receiver", To: "Logger", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"}},
				{
					{From: "Receiver", To: "Sender", EventType: "AckEvent", HandlerType: "OnEvent", HandlerEventType: "PingEvent"},
					{From: "Sender", To: "Receiver", EventType: "NotifyEvent", HandlerType: "OnEvent", HandlerEventType: "AckEvent"},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("buildElements mismatch (-want +got):\n%s", diff)
	}
}
