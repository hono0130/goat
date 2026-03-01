package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMeetingRoomReservationWithoutExclusion(t *testing.T) {
	opts := createMeetingRoomWithoutExclusionModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	sm := func(name, state, details string) goat.StateMachineSnapshot {
		return goat.StateMachineSnapshot{Name: name, State: state, Details: details}
	}
	ev := func(target, event, details string) goat.EventSnapshot {
		return goat.EventSnapshot{TargetMachine: target, EventName: event, Details: details}
	}

	client0Idle := sm("ClientStateMachine", "{Name:StateType,Type:main.StateType,Value:ClientIdle}", "{Name:ClientID,Type:int,Value:0},{Name:TargetRoom,Type:int,Value:101}")
	client1Idle := sm("ClientStateMachine", "{Name:StateType,Type:main.StateType,Value:ClientIdle}", "{Name:ClientID,Type:int,Value:1},{Name:TargetRoom,Type:int,Value:101}")
	dbIdle := sm("DBStateMachine", "{Name:StateType,Type:main.StateType,Value:DBIdle}", "{Name:Reservations,Type:[]main.Reservation,Value:[]}")
	dbIdleReserved0 := sm("DBStateMachine", "{Name:StateType,Type:main.StateType,Value:DBIdle}", "{Name:Reservations,Type:[]main.Reservation,Value:[{101 0}]}")
	dbIdleReservedBoth := sm("DBStateMachine", "{Name:StateType,Type:main.StateType,Value:DBIdle}", "{Name:Reservations,Type:[]main.Reservation,Value:[{101 0} {101 1}]}")
	server1Idle := sm("ServerStateMachine", "{Name:StateType,Type:main.StateType,Value:ServerIdle}", "no fields")
	server2Idle := sm("ServerStateMachine", "{Name:StateType,Type:main.StateType,Value:ServerIdle}", "no fields")
	server1Processing := sm("ServerStateMachine", "{Name:StateType,Type:main.StateType,Value:ServerProcessing}", "no fields")
	server2Processing := sm("ServerStateMachine", "{Name:StateType,Type:main.StateType,Value:ServerProcessing}", "no fields")

	entryClient := func() goat.EventSnapshot { return ev("ClientStateMachine", "entryEvent", "no fields") }
	exitClient := func() goat.EventSnapshot { return ev("ClientStateMachine", "exitEvent", "no fields") }
	transClient := func() goat.EventSnapshot {
		return ev("ClientStateMachine", "transitionEvent", "{Name:To,Type:goat.AbstractState,Value:&{{0} ClientRequesting}}")
	}
	entryDB := func() goat.EventSnapshot { return ev("DBStateMachine", "entryEvent", "no fields") }
	entryServer := func() goat.EventSnapshot { return ev("ServerStateMachine", "entryEvent", "no fields") }
	exitServer := func() goat.EventSnapshot { return ev("ServerStateMachine", "exitEvent", "no fields") }
	transServer := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "transitionEvent", "{Name:To,Type:goat.AbstractState,Value:&{{0} ServerProcessing}}")
	}
	reserveReq0 := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "ReservationRequestEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:0}")
	}
	reserveReq1 := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "ReservationRequestEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:1}")
	}
	dbSelect1 := func() goat.EventSnapshot {
		return ev("DBStateMachine", "DBSelectEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:1}")
	}
	dbSelectResult0NotReserved := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "DBSelectResultEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:0},{Name:IsReserved,Type:bool,Value:false}")
	}
	dbSelectResult1NotReserved := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "DBSelectResultEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:1},{Name:IsReserved,Type:bool,Value:false}")
	}
	dbUpdate0 := func() goat.EventSnapshot {
		return ev("DBStateMachine", "DBUpdateEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:0}")
	}
	dbUpdate1 := func() goat.EventSnapshot {
		return ev("DBStateMachine", "DBUpdateEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:1}")
	}
	dbUpdateResult0Success := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "DBUpdateResultEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:0},{Name:Succeeded,Type:bool,Value:true}")
	}
	dbUpdateResult1Success := func() goat.EventSnapshot {
		return ev("ServerStateMachine", "DBUpdateResultEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:1},{Name:Succeeded,Type:bool,Value:true}")
	}

	expected := &goat.Result{
		Violations: []goat.Violation{
			{
				Rule: "Always no-double-book",
				Path: []goat.WorldSnapshot{
					// [0] initial state
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents:  []goat.EventSnapshot{entryClient(), entryClient(), entryDB(), entryServer(), entryServer()},
					},
					// [1] client0 transitions to ClientRequesting
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents:  []goat.EventSnapshot{exitClient(), transClient(), entryClient(), entryClient(), entryDB(), entryServer(), reserveReq0(), entryServer()},
					},
					// [2] client1 also transitions to ClientRequesting
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents:  []goat.EventSnapshot{exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(), entryDB(), entryServer(), reserveReq0(), entryServer(), reserveReq1()},
					},
					// [3]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents:  []goat.EventSnapshot{exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(), entryServer(), reserveReq0(), entryServer(), reserveReq1()},
					},
					// [4]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents:  []goat.EventSnapshot{exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(), reserveReq0(), entryServer(), reserveReq1()},
					},
					// [5] server1 starts processing client0's request
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							ev("DBStateMachine", "DBSelectEvent", "{Name:RoomID,Type:int,Value:101},{Name:ClientID,Type:int,Value:0}"),
							exitServer(), transServer(), entryServer(),
							entryServer(), reserveReq1(),
						},
					},
					// [6]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							exitServer(), transServer(), entryServer(),
							dbSelectResult0NotReserved(),
							entryServer(), reserveReq1(),
						},
					},
					// [7]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Idle, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							transServer(), entryServer(),
							dbSelectResult0NotReserved(),
							entryServer(), reserveReq1(),
						},
					},
					// [8] server1 is now processing
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							entryServer(),
							dbSelectResult0NotReserved(),
							entryServer(), reserveReq1(),
						},
					},
					// [9]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbSelectResult0NotReserved(),
							entryServer(), reserveReq1(),
						},
					},
					// [10] server1 issues DBUpdate for client0
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdate0(),
							entryServer(), reserveReq1(),
						},
					},
					// [11]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdate0(),
							reserveReq1(),
						},
					},
					// [12] server2 starts processing client1
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbSelect1(),
							dbUpdate0(),
							exitServer(), transServer(), entryServer(),
						},
					},
					// [13]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdle, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdate0(),
							exitServer(), transServer(), entryServer(),
							dbSelectResult1NotReserved(),
						},
					},
					// [14] DB now has reservation for client0
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReserved0, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdateResult0Success(),
							exitServer(), transServer(), entryServer(),
							dbSelectResult1NotReserved(),
						},
					},
					// [15]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReserved0, server1Processing, server2Idle},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdateResult0Success(),
							transServer(), entryServer(),
							dbSelectResult1NotReserved(),
						},
					},
					// [16] both servers processing
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReserved0, server1Processing, server2Processing},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdateResult0Success(),
							entryServer(),
							dbSelectResult1NotReserved(),
						},
					},
					// [17]
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReserved0, server1Processing, server2Processing},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdateResult0Success(),
							dbSelectResult1NotReserved(),
						},
					},
					// [18] server2 issues DBUpdate for client1
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReserved0, server1Processing, server2Processing},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdate1(),
							dbUpdateResult0Success(),
						},
					},
					// [19] double booking!
					{
						StateMachines: []goat.StateMachineSnapshot{client0Idle, client1Idle, dbIdleReservedBoth, server1Processing, server2Processing},
						QueuedEvents: []goat.EventSnapshot{
							exitClient(), transClient(), entryClient(), exitClient(), transClient(), entryClient(),
							dbUpdateResult0Success(),
							dbUpdateResult1Success(),
						},
					},
				},
			},
		},
		Summary: goat.Summary{TotalWorlds: 12152},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreUnexported(goat.Result{}),
		cmpopts.IgnoreFields(goat.Summary{}, "ExecutionTimeMs"),
	}
	if diff := cmp.Diff(expected, result, cmpOpts...); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
