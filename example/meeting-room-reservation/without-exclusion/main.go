package main

import (
	"context"
	"fmt"

	"github.com/goatx/goat"
	"github.com/google/uuid"
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
	goat.Event
	RoomID   int
	ClientID int
	Client   *ClientStateMachine
}

type ReservationResultEvent struct {
	goat.Event
	RoomID    int
	ClientID  int
	Succeeded bool
}

type DBSelectEvent struct {
	goat.Event
	RoomID   int
	ClientID int
	Server   *ServerStateMachine
}

type DBSelectResultEvent struct {
	goat.Event
	RoomID     int
	ClientID   int
	IsReserved bool
}

type DBUpdateEvent struct {
	goat.Event
	RoomID   int
	ClientID int
	Server   *ServerStateMachine
}

type DBUpdateResultEvent struct {
	goat.Event
	RoomID    int
	ClientID  int
	Succeeded bool
}

type Reservation struct {
	UUID     string
	RoomID   int
	ClientID int
}

type ServerStateMachine struct {
	goat.StateMachine
	CurrentRequest *ReservationRequestEvent
	DB             *DBStateMachine
}

type DBStateMachine struct {
	goat.StateMachine
	Reservations []Reservation
}

type ClientStateMachine struct {
	goat.StateMachine
	ClientID   int
	TargetRoom int
	Server     *ServerStateMachine
}

func createMeetingRoomWithoutExclusionModel() []goat.Option {
	// === Database Spec ===
	dbSpec := goat.NewStateMachineSpec(&DBStateMachine{})
	dbIdle := &State{StateType: DBIdle}

	dbSpec.DefineStates(dbIdle).SetInitialState(dbIdle)

	// DBのハンドラーをSpecに登録
	goat.OnEvent(dbSpec, dbIdle, &DBSelectEvent{},
		func(ctx context.Context, event *DBSelectEvent, db *DBStateMachine) {
			// Check if room is already reserved
			isReserved := false
			for _, res := range db.Reservations {
				if res.RoomID == event.RoomID {
					isReserved = true
					break
				}
			}

			resultEvent := &DBSelectResultEvent{
				RoomID:     event.RoomID,
				ClientID:   event.ClientID,
				IsReserved: isReserved,
			}

			goat.SendTo(ctx, event.Server, resultEvent)
		})

	goat.OnEvent(dbSpec, dbIdle, &DBUpdateEvent{},
		func(ctx context.Context, event *DBUpdateEvent, db *DBStateMachine) {
			db.Reservations = append(db.Reservations, Reservation{
				UUID:     uuid.New().String(),
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
			})

			resultEvent := &DBUpdateResultEvent{
				RoomID:    event.RoomID,
				ClientID:  event.ClientID,
				Succeeded: true,
			}

			goat.SendTo(ctx, event.Server, resultEvent)
		})

	// === Server Spec ===
	serverSpec := goat.NewStateMachineSpec(&ServerStateMachine{})
	serverIdle := &State{StateType: ServerIdle}
	serverProcessing := &State{StateType: ServerProcessing}

	serverSpec.DefineStates(serverIdle, serverProcessing).SetInitialState(serverIdle)

	goat.OnEvent(serverSpec, serverIdle, &ReservationRequestEvent{},
		func(ctx context.Context, event *ReservationRequestEvent, server *ServerStateMachine) {
			server.CurrentRequest = event

			// First, SELECT to check if room is already reserved
			selectEvent := &DBSelectEvent{
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
				Server:   server,
			}

			goat.SendTo(ctx, server.DB, selectEvent)
			goat.Goto(ctx, serverProcessing)
		})

	// Handle SELECT result
	goat.OnEvent(serverSpec, serverProcessing, &DBSelectResultEvent{},
		func(ctx context.Context, event *DBSelectResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			if event.IsReserved {
				// Room is already reserved, send failure to client
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: false,
				}

				goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
				server.CurrentRequest = nil
				goat.Goto(ctx, serverIdle)
				return
			}

			// Room is not reserved, proceed with UPDATE
			updateEvent := &DBUpdateEvent{
				RoomID:   server.CurrentRequest.RoomID,
				ClientID: server.CurrentRequest.ClientID,
				Server:   server,
			}

			goat.SendTo(ctx, server.DB, updateEvent)
		})

	goat.OnEvent(serverSpec, serverProcessing, &DBUpdateResultEvent{},
		func(ctx context.Context, event *DBUpdateResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			resultEvent := &ReservationResultEvent{
				RoomID:    server.CurrentRequest.RoomID,
				ClientID:  server.CurrentRequest.ClientID,
				Succeeded: event.Succeeded,
			}

			goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
			server.CurrentRequest = nil
			goat.Goto(ctx, serverIdle)
		})

	// === Client Spec ===
	clientSpec := goat.NewStateMachineSpec(&ClientStateMachine{})
	clientIdle := &State{StateType: ClientIdle}
	clientRequesting := &State{StateType: ClientRequesting}
	clientEnd := &State{StateType: ClientEnd}

	clientSpec.DefineStates(clientIdle, clientRequesting, clientEnd).SetInitialState(clientIdle)

	goat.OnEntry(clientSpec, clientIdle,
		func(ctx context.Context, client *ClientStateMachine) {
			requestEvent := &ReservationRequestEvent{
				RoomID:   client.TargetRoom,
				ClientID: client.ClientID,
				Client:   client,
			}

			goat.SendTo(ctx, client.Server, requestEvent)
			goat.Goto(ctx, clientRequesting)
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

	goat.OnEntry(clientSpec, clientEnd,
		func(ctx context.Context, client *ClientStateMachine) {
		})

	// === Create Instances ===
	// Create database instance
	db := dbSpec.NewInstance()
	db.Reservations = make([]Reservation, 0)

	// Create server instances
	server1 := serverSpec.NewInstance()
	server1.DB = db

	server2 := serverSpec.NewInstance()
	server2.DB = db

	// Create client instances
	client1 := clientSpec.NewInstance()
	client1.ClientID = 0
	client1.TargetRoom = 101
	client1.Server = server1

	client2 := clientSpec.NewInstance()
	client2.ClientID = 1
	client2.TargetRoom = 101
	client2.Server = server2

	opts := []goat.Option{
		goat.WithStateMachines(server1, server2, db, client1, client2),
		goat.WithInvariants(
			// Invariant: A room should not be reserved by multiple clients
			goat.NewInvariant(db, func(db *DBStateMachine) bool {
				roomClients := make(map[int]map[int]bool)
				for _, res := range db.Reservations {
					if _, ok := roomClients[res.RoomID]; !ok {
						roomClients[res.RoomID] = make(map[int]bool)
					}
					roomClients[res.RoomID][res.ClientID] = true
				}

				for _, clients := range roomClients {
					if len(clients) > 1 {
						return false
					}
				}

				return true
			}),
		),
	}

	return opts
}

func main() {
	fmt.Println("Meeting Room Reservation System (Without Proper Exclusion Control)")
	fmt.Println("Simulating: SELECT → UPDATE (without locking)")

	opts := createMeetingRoomWithoutExclusionModel()

	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
