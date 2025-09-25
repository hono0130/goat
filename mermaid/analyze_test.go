package mermaid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestStateMachineOrder(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	got := stateMachineOrder(&pkg)
	want := []string{"Sender", "Receiver", "Logger"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("stateMachineOrder mismatch (-want +got):\n%s", diff)
	}
}

func TestCommunicationFlows(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	got := communicationFlows(&pkg)
	want := []flow{
		{from: "Sender", to: "Receiver", eventType: "PingEvent", handlerType: "OnEntry"},
		{from: "Receiver", to: "Sender", eventType: "AckEvent", handlerType: "OnEvent", handlerEventType: "PingEvent"},
		{from: "Receiver", to: "Logger", eventType: "NotifyEvent", handlerType: "OnEvent", handlerEventType: "PingEvent"},
		{from: "Sender", to: "Receiver", eventType: "NotifyEvent", handlerType: "OnEvent", handlerEventType: "AckEvent"},
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(flow{}), cmpopts.IgnoreFields(flow{}, "handlerID")); diff != "" {
		t.Fatalf("communicationFlows mismatch (-want +got):\n%s", diff)
	}
	for i, f := range got {
		if f.handlerID == "" {
			t.Fatalf("flow[%d] handlerID is empty", i)
		}
	}
}

func TestBuildElements(t *testing.T) {
	pkg := loadWorkflowPackage(t)
	flows := communicationFlows(&pkg)
	got := buildElements(flows)
	want := []element{
		{
			flows: []flow{{from: "Sender", to: "Receiver", eventType: "PingEvent", handlerType: "OnEntry"}},
		},
		{
			flows: nil,
			branches: [][]flow{
				{{from: "Receiver", to: "Logger", eventType: "NotifyEvent", handlerType: "OnEvent", handlerEventType: "PingEvent"}},
				{
					{from: "Receiver", to: "Sender", eventType: "AckEvent", handlerType: "OnEvent", handlerEventType: "PingEvent"},
					{from: "Sender", to: "Receiver", eventType: "NotifyEvent", handlerType: "OnEvent", handlerEventType: "AckEvent"},
				},
			},
		},
	}
	if diff := cmp.Diff(
		want,
		got,
		cmp.AllowUnexported(element{}, flow{}),
		cmpopts.IgnoreFields(flow{}, "handlerID"),
	); diff != "" {
		t.Fatalf("buildElements mismatch (-want +got):\n%s", diff)
	}
}
