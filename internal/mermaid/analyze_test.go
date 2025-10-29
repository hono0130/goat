package mermaid

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/goatx/goat-cli/internal/load"
	"github.com/goatx/goat-cli/internal/test"
)

func loadSpecPackage(t *testing.T) *load.PackageInfo {
	t.Helper()
	pkg, err := load.Load(test.FixtureDir(t))
	if err != nil {
		t.Fatalf("failed to load fixture package: %v", err)
	}
	return pkg
}

func TestStateMachineOrder(t *testing.T) {
	t.Parallel()
	pkg := loadSpecPackage(t)
	got, err := stateMachineOrder(pkg)
	if err != nil {
		t.Fatalf("stateMachineOrder returned error: %v", err)
	}
	want := []string{
		"ClientStateMachine",
		"ServerStateMachine",
		"DBStateMachine",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("stateMachineOrder mismatch (-want +got):\n%s", diff)
	}
}

func TestCommunicationFlows(t *testing.T) {
	t.Parallel()
	pkg := loadSpecPackage(t)
	got, err := communicationFlows(pkg)
	if err != nil {
		t.Fatalf("communicationFlows returned error: %v", err)
	}
	want := []flow{
		{
			from:             "ClientStateMachine",
			to:               "ServerStateMachine",
			eventType:        "ReservationRequestEvent",
			handlerType:      onEntryHandler,
			handlerEventType: "",
			handlerID:        "ClientStateMachine_OnEntry__spec.go:127",
			fileName:         "spec.go",
			line:             133,
		},
		{
			from:             "ServerStateMachine",
			to:               "DBStateMachine",
			eventType:        "DBSelectEvent",
			handlerType:      onEventHandler,
			handlerEventType: "ReservationRequestEvent",
			handlerID:        "ServerStateMachine_OnEvent_ReservationRequestEvent_spec.go:138",
			fileName:         "spec.go",
			line:             146,
		},
		{
			from:             "DBStateMachine",
			to:               "ServerStateMachine",
			eventType:        "DBSelectResultEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBSelectEvent",
			handlerID:        "DBStateMachine_OnEvent_DBSelectEvent_spec.go:151",
			fileName:         "spec.go",
			line:             173,
		},
		{
			from:             "ServerStateMachine",
			to:               "DBStateMachine",
			eventType:        "DBUpdateEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBSelectResultEvent",
			handlerID:        "ServerStateMachine_OnEvent_DBSelectResultEvent_spec.go:177",
			fileName:         "spec.go",
			line:             190,
		},
		{
			from:             "ServerStateMachine",
			to:               "ClientStateMachine",
			eventType:        "ReservationResultEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBSelectResultEvent",
			handlerID:        "ServerStateMachine_OnEvent_DBSelectResultEvent_spec.go:177",
			fileName:         "spec.go",
			line:             198,
		},
		{
			from:             "DBStateMachine",
			to:               "ServerStateMachine",
			eventType:        "DBUpdateResultEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBUpdateEvent",
			handlerID:        "DBStateMachine_OnEvent_DBUpdateEvent_spec.go:216",
			fileName:         "spec.go",
			line:             237,
		},
		{
			from:             "ServerStateMachine",
			to:               "ClientStateMachine",
			eventType:        "ReservationResultEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBUpdateResultEvent",
			handlerID:        "ServerStateMachine_OnEvent_DBUpdateResultEvent_spec.go:241",
			fileName:         "spec.go",
			line:             254,
		},
		{
			from:             "ServerStateMachine",
			to:               "ClientStateMachine",
			eventType:        "ReservationRetryEvent",
			handlerType:      onEventHandler,
			handlerEventType: "DBUpdateResultEvent",
			handlerID:        "ServerStateMachine_OnEvent_DBUpdateResultEvent_spec.go:241",
			fileName:         "spec.go",
			line:             261,
		},
		{
			from:             "ClientStateMachine",
			to:               "ServerStateMachine",
			eventType:        "ReservationRequestEvent",
			handlerType:      onEventHandler,
			handlerEventType: "ReservationRetryEvent",
			handlerID:        "ClientStateMachine_OnEvent_ReservationRetryEvent_spec.go:280",
			fileName:         "spec.go",
			line:             287,
		},
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(flow{})); diff != "" {
		t.Fatalf("communicationFlows mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildElements(t *testing.T) {
	t.Parallel()
	pkg := loadSpecPackage(t)
	flows, err := communicationFlows(pkg)
	if err != nil {
		t.Fatalf("communicationFlows returned error: %v", err)
	}

	got := buildElements(flows)
	want := []element{
		{
			flow: flow{
				from:             "ClientStateMachine",
				to:               "ServerStateMachine",
				eventType:        "ReservationRequestEvent",
				handlerType:      onEntryHandler,
				handlerEventType: "",
				handlerID:        "ClientStateMachine_OnEntry__spec.go:127",
				fileName:         "spec.go",
				line:             133,
			},
		},
		{
			flow: flow{
				from:             "ServerStateMachine",
				to:               "DBStateMachine",
				eventType:        "DBSelectEvent",
				handlerType:      onEventHandler,
				handlerEventType: "ReservationRequestEvent",
				handlerID:        "ServerStateMachine_OnEvent_ReservationRequestEvent_spec.go:138",
				fileName:         "spec.go",
				line:             146,
			},
		},
		{
			flow: flow{
				from:             "DBStateMachine",
				to:               "ServerStateMachine",
				eventType:        "DBSelectResultEvent",
				handlerType:      onEventHandler,
				handlerEventType: "DBSelectEvent",
				handlerID:        "DBStateMachine_OnEvent_DBSelectEvent_spec.go:151",
				fileName:         "spec.go",
				line:             173,
			},
		},
		{
			branches: []branch{
				{
					flow: flow{
						from:             "ServerStateMachine",
						to:               "ClientStateMachine",
						eventType:        "ReservationResultEvent",
						handlerType:      onEventHandler,
						handlerEventType: "DBSelectResultEvent",
						handlerID:        "ServerStateMachine_OnEvent_DBSelectResultEvent_spec.go:177",
						fileName:         "spec.go",
						line:             198,
					},
				},
				{
					flow: flow{
						from:             "ServerStateMachine",
						to:               "DBStateMachine",
						eventType:        "DBUpdateEvent",
						handlerType:      onEventHandler,
						handlerEventType: "DBSelectResultEvent",
						handlerID:        "ServerStateMachine_OnEvent_DBSelectResultEvent_spec.go:177",
						fileName:         "spec.go",
						line:             190,
					},
					elements: []element{
						{
							flow: flow{
								from:             "DBStateMachine",
								to:               "ServerStateMachine",
								eventType:        "DBUpdateResultEvent",
								handlerType:      onEventHandler,
								handlerEventType: "DBUpdateEvent",
								handlerID:        "DBStateMachine_OnEvent_DBUpdateEvent_spec.go:216",
								fileName:         "spec.go",
								line:             237,
							},
						},
						{
							branches: []branch{
								{
									flow: flow{
										from:             "ServerStateMachine",
										to:               "ClientStateMachine",
										eventType:        "ReservationResultEvent",
										handlerType:      onEventHandler,
										handlerEventType: "DBUpdateResultEvent",
										handlerID:        "ServerStateMachine_OnEvent_DBUpdateResultEvent_spec.go:241",
										fileName:         "spec.go",
										line:             254,
									},
								},
								{
									flow: flow{
										from:             "ServerStateMachine",
										to:               "ClientStateMachine",
										eventType:        "ReservationRetryEvent",
										handlerType:      onEventHandler,
										handlerEventType: "DBUpdateResultEvent",
										handlerID:        "ServerStateMachine_OnEvent_DBUpdateResultEvent_spec.go:241",
										fileName:         "spec.go",
										line:             261,
									},
									elements: []element{
										{
											flow: flow{
												from:             "ClientStateMachine",
												to:               "ServerStateMachine",
												eventType:        "ReservationRequestEvent",
												handlerType:      onEventHandler,
												handlerEventType: "ReservationRetryEvent",
												handlerID:        "ClientStateMachine_OnEvent_ReservationRetryEvent_spec.go:280",
												fileName:         "spec.go",
												line:             287,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(flow{}, element{}, branch{})); diff != "" {
		t.Fatalf("buildElements mismatch (-want +got):\n%s", diff)
	}
}
