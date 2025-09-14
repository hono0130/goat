package goat

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOrderedParticipants(t *testing.T) {
	elements := []SequenceDiagramElement{
		{
			Flows: []CommunicationFlow{{From: "B", To: "D", EventType: "E3"}},
		},
		{
			Flows: []CommunicationFlow{{From: "C", To: "A", EventType: "E2"}},
		},
	}
	order := []string{"A", "B"}

	got := orderedParticipants(elements, order)
	want := []string{"A", "B", "C", "D"}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("orderedParticipants mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildSequenceDiagramElements(t *testing.T) {
	flows := []CommunicationFlow{
		{From: "A", To: "B", EventType: "E1", HandlerType: onEntryHandler, HandlerID: "A_OnEntry"},
		{From: "B", To: "C", EventType: "E2", HandlerType: onEventHandler, HandlerEventType: "E1", HandlerID: "B_OnEvent_E1"},
		{From: "B", To: "D", EventType: "E3", HandlerType: onEventHandler, HandlerEventType: "E1", HandlerID: "B_OnEvent_E1"},
		{From: "C", To: "E", EventType: "E4", HandlerType: onEventHandler, HandlerEventType: "E2", HandlerID: "C_OnEvent_E2"},
	}

	got := buildSequenceDiagramElements(flows)
	want := []SequenceDiagramElement{
		{Flows: []CommunicationFlow{{From: "A", To: "B", EventType: "E1", HandlerType: onEntryHandler, HandlerID: "A_OnEntry"}}},
		{Flows: []CommunicationFlow{{From: "B", To: "D", EventType: "E3", HandlerType: onEventHandler, HandlerEventType: "E1", HandlerID: "B_OnEvent_E1"}}, IsOptional: true},
		{Flows: []CommunicationFlow{
			{From: "B", To: "C", EventType: "E2", HandlerType: onEventHandler, HandlerEventType: "E1", HandlerID: "B_OnEvent_E1"},
			{From: "C", To: "E", EventType: "E4", HandlerType: onEventHandler, HandlerEventType: "E2", HandlerID: "C_OnEvent_E2"},
		}, IsOptional: true},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("buildSequenceDiagramElements mismatch (-want +got):\n%s", diff)
	}
}
