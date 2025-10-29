package main

import (
	"context"

	"github.com/goatx/goat"
)

type (
	StateType string
)

const (
	// Server states
	ServerIdle       StateType = "ServerIdle"
	ServerProcessing StateType = "ServerProcessing"

	// Database states
	DBIdle StateType = "DBIdle"

	// Client states
	ClientIdle       StateType = "ClientIdle"
	ClientRequesting StateType = "ClientRequesting"
	ClientEnd        StateType = "ClientEnd"
)

type State struct {
	goat.State
	StateType StateType
}

type ReservationRequestEvent struct {
	goat.Event[*ClientStateMachine, *ServerStateMachine]
	RoomID   int
	ClientID int
}

type ReservationResultEvent struct {
	goat.Event[*ServerStateMachine, *ClientStateMachine]
	RoomID    int
	ClientID  int
	Succeeded bool
}

type ReservationRetryEvent struct {
	goat.Event[*ServerStateMachine, *ClientStateMachine]
	RoomID   int
	ClientID int
}

type DBSelectEvent struct {
	goat.Event[*ServerStateMachine, *DBStateMachine]
	RoomID   int
	ClientID int
}

type DBSelectResultEvent struct {
	goat.Event[*DBStateMachine, *ServerStateMachine]
	RoomID     int
	ClientID   int
	IsReserved bool
	IsLocked   bool
}

type DBUpdateEvent struct {
	goat.Event[*ServerStateMachine, *DBStateMachine]
	RoomID   int
	ClientID int
}

type DBUpdateResultEvent struct {
	goat.Event[*DBStateMachine, *ServerStateMachine]
	RoomID    int
	ClientID  int
	Succeeded bool
}

type Reservation struct {
	RoomID   int
	ClientID int
}

type ClientStateMachine struct {
	goat.StateMachine
	ClientID   int
	TargetRoom int
	Server     *ServerStateMachine
}

type ServerStateMachine struct {
	goat.StateMachine
	CurrentRequest *ReservationRequestEvent
	DB             *DBStateMachine
}

type DBStateMachine struct {
	goat.StateMachine
	Reservations []Reservation
	LockedRooms  map[int]int
}

func createMeetingRoomWithExclusionModel() {
	// === Client Spec ===
	clientSpec := goat.NewStateMachineSpec(&ClientStateMachine{})
	clientIdle := &State{StateType: ClientIdle}
	clientRequesting := &State{StateType: ClientRequesting}
	clientEnd := &State{StateType: ClientEnd}

	clientSpec.DefineStates(clientIdle, clientRequesting, clientEnd).SetInitialState(clientIdle)

	// === Server Spec ===
	serverSpec := goat.NewStateMachineSpec(&ServerStateMachine{})
	serverIdle := &State{StateType: ServerIdle}
	serverProcessing := &State{StateType: ServerProcessing}

	serverSpec.DefineStates(serverIdle, serverProcessing).SetInitialState(serverIdle)

	// === Database Spec ===
	dbSpec := goat.NewStateMachineSpec(&DBStateMachine{})
	dbIdle := &State{StateType: DBIdle}

	dbSpec.DefineStates(dbIdle).SetInitialState(dbIdle)

	// === Handlers ===

	goat.OnEntry(clientSpec, clientIdle,
		func(ctx context.Context, client *ClientStateMachine) {
			requestEvent := &ReservationRequestEvent{
				RoomID:   client.TargetRoom,
				ClientID: client.ClientID,
			}

			goat.SendTo(ctx, client.Server, requestEvent)
			goat.Goto(ctx, clientRequesting)
		})

	goat.OnEvent(serverSpec, serverIdle, &ReservationRequestEvent{},
		func(ctx context.Context, event *ReservationRequestEvent, server *ServerStateMachine) {
			server.CurrentRequest = event

			selectEvent := &DBSelectEvent{
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
			}

			goat.SendTo(ctx, server.DB, selectEvent)
			goat.Goto(ctx, serverProcessing)
		})

	goat.OnEvent(dbSpec, dbIdle, &DBSelectEvent{},
		func(ctx context.Context, event *DBSelectEvent, db *DBStateMachine) {
			isReserved := false
			for _, res := range db.Reservations {
				if res.RoomID == event.RoomID {
					isReserved = true
					break
				}
			}

			isLocked := false
			if _, exists := db.LockedRooms[event.RoomID]; !exists {
				db.LockedRooms[event.RoomID] = event.ClientID
				isLocked = true
			}

			resultEvent := &DBSelectResultEvent{
				RoomID:     event.RoomID,
				ClientID:   event.ClientID,
				IsReserved: isReserved,
				IsLocked:   isLocked,
			}

			goat.SendTo(ctx, event.Sender(), resultEvent)
		})

	goat.OnEvent(serverSpec, serverProcessing, &DBSelectResultEvent{},
		func(ctx context.Context, event *DBSelectResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			if event.IsLocked {
				if !event.IsReserved {
					updateEvent := &DBUpdateEvent{
						RoomID:   server.CurrentRequest.RoomID,
						ClientID: server.CurrentRequest.ClientID,
					}

					goat.SendTo(ctx, server.DB, updateEvent)
				} else {
					resultEvent := &ReservationResultEvent{
						RoomID:    server.CurrentRequest.RoomID,
						ClientID:  server.CurrentRequest.ClientID,
						Succeeded: false,
					}

					goat.SendTo(ctx, server.CurrentRequest.Sender(), resultEvent)
					server.CurrentRequest = nil
					goat.Goto(ctx, serverIdle)
				}
			} else {
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: false,
				}

				goat.SendTo(ctx, server.CurrentRequest.Sender(), resultEvent)
				server.CurrentRequest = nil
				goat.Goto(ctx, serverIdle)
			}
		})

	goat.OnEvent(dbSpec, dbIdle, &DBUpdateEvent{},
		func(ctx context.Context, event *DBUpdateEvent, db *DBStateMachine) {
			hasLock := false
			if clientID, exists := db.LockedRooms[event.RoomID]; exists && clientID == event.ClientID {
				hasLock = true
			}

			succeeded := false
			if hasLock {
				db.Reservations = append(db.Reservations, Reservation{
					RoomID:   event.RoomID,
					ClientID: event.ClientID,
				})
				succeeded = true
			}

			resultEvent := &DBUpdateResultEvent{
				RoomID:    event.RoomID,
				ClientID:  event.ClientID,
				Succeeded: succeeded,
			}

			goat.SendTo(ctx, event.Sender(), resultEvent)
		})

	goat.OnEvent(serverSpec, serverProcessing, &DBUpdateResultEvent{},
		func(ctx context.Context, event *DBUpdateResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			if event.Succeeded {
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: true,
				}

				goat.SendTo(ctx, server.CurrentRequest.Sender(), resultEvent)
			} else {
				retryEvent := &ReservationRetryEvent{
					RoomID:   server.CurrentRequest.RoomID,
					ClientID: server.CurrentRequest.ClientID,
				}

				goat.SendTo(ctx, server.CurrentRequest.Sender(), retryEvent)
			}

			server.CurrentRequest = nil
			goat.Goto(ctx, serverIdle)
		})

	goat.OnEvent(clientSpec, clientRequesting, &ReservationResultEvent{},
		func(ctx context.Context, event *ReservationResultEvent, client *ClientStateMachine) {
			if event.ClientID == client.ClientID {
				if event.Succeeded {
					goat.Goto(ctx, clientEnd)
				} else {
					goat.Goto(ctx, clientEnd)
				}
			}
		})

	goat.OnEvent(clientSpec, clientRequesting, &ReservationRetryEvent{},
		func(ctx context.Context, event *ReservationRetryEvent, client *ClientStateMachine) {
			if event.ClientID == client.ClientID {
				requestEvent := &ReservationRequestEvent{
					RoomID:   client.TargetRoom,
					ClientID: client.ClientID,
				}

				goat.SendTo(ctx, event.Sender(), requestEvent)
			}
		})

	goat.OnEntry(clientSpec, clientEnd,
		func(ctx context.Context, client *ClientStateMachine) {
		})

}

func main() {
	createMeetingRoomWithExclusionModel()
}
