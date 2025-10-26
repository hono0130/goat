package main

import (
	"context"
	"log"

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

	goat.OnEvent(dbSpec, dbIdle, &DBSelectEvent{},
		func(ctx context.Context, event *DBSelectEvent, db *DBStateMachine) {
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

			goat.SendTo(ctx, event.Sender(), resultEvent)
		})

	goat.OnEvent(dbSpec, dbIdle, &DBUpdateEvent{},
		func(ctx context.Context, event *DBUpdateEvent, db *DBStateMachine) {
			db.Reservations = append(db.Reservations, Reservation{
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
			})

			resultEvent := &DBUpdateResultEvent{
				RoomID:    event.RoomID,
				ClientID:  event.ClientID,
				Succeeded: true,
			}

			goat.SendTo(ctx, event.Sender(), resultEvent)
		})

	// === Server Spec ===
	serverSpec := goat.NewStateMachineSpec(&ServerStateMachine{})
	serverIdle := &State{StateType: ServerIdle}
	serverProcessing := &State{StateType: ServerProcessing}

	serverSpec.DefineStates(serverIdle, serverProcessing).SetInitialState(serverIdle)

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

	goat.OnEvent(serverSpec, serverProcessing, &DBSelectResultEvent{},
		func(ctx context.Context, event *DBSelectResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			if event.IsReserved {
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: false,
				}

				goat.SendTo(ctx, server.CurrentRequest.Sender(), resultEvent)
				server.CurrentRequest = nil
				goat.Goto(ctx, serverIdle)
				return
			}

			updateEvent := &DBUpdateEvent{
				RoomID:   server.CurrentRequest.RoomID,
				ClientID: server.CurrentRequest.ClientID,
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

			goat.SendTo(ctx, server.CurrentRequest.Sender(), resultEvent)
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
	db, err := dbSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	db.Reservations = make([]Reservation, 0)

	// Create server instances
	server1, err := serverSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	server1.DB = db

	server2, err := serverSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	server2.DB = db

	// Create client instances
	client1, err := clientSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client1.ClientID = 0
	client1.TargetRoom = 101
	client1.Server = server1

	client2, err := clientSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client2.ClientID = 1
	client2.TargetRoom = 101
	client2.Server = server2

	cond := goat.NewCondition("no-double-book", db, func(db *DBStateMachine) bool {
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
	})

	opts := []goat.Option{
		goat.WithStateMachines(server1, server2, db, client1, client2),
		goat.WithConditions(cond),
		goat.WithInvariants(cond),
	}

	return opts
}

func main() {
	opts := createMeetingRoomWithoutExclusionModel()
	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
