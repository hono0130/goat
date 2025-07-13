package main

import (
	"context"
	"fmt"
	"os"

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

func (sm *ServerStateMachine) NewMachine(db *DBStateMachine) {
	var (
		idle       = &State{StateType: ServerIdle}
		processing = &State{StateType: ServerProcessing}
	)

	sm.StateMachine.New(idle, processing)
	sm.SetInitialState(idle)
	sm.DB = db

	goat.OnEvent(sm, idle, &ReservationRequestEvent{},
		func(ctx context.Context, event *ReservationRequestEvent, server *ServerStateMachine) {
			server.CurrentRequest = event

			// First, SELECT to check if room is already reserved
			selectEvent := &DBSelectEvent{
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
				Server:   server,
			}

			goat.SendTo(ctx, server.DB, selectEvent)
			goat.Goto(ctx, processing)
		},
	)

	// Handle SELECT result
	goat.OnEvent(sm, processing, &DBSelectResultEvent{},
		func(ctx context.Context, event *DBSelectResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, idle)
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
				goat.Goto(ctx, idle)
				return
			}

			// Room is not reserved, proceed with UPDATE
			updateEvent := &DBUpdateEvent{
				RoomID:   server.CurrentRequest.RoomID,
				ClientID: server.CurrentRequest.ClientID,
				Server:   server,
			}

			goat.SendTo(ctx, server.DB, updateEvent)
		},
	)

	goat.OnEvent(sm, processing, &DBUpdateResultEvent{},
		func(ctx context.Context, event *DBUpdateResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, idle)
				return
			}

			resultEvent := &ReservationResultEvent{
				RoomID:    server.CurrentRequest.RoomID,
				ClientID:  server.CurrentRequest.ClientID,
				Succeeded: event.Succeeded,
			}

			goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
			server.CurrentRequest = nil
			goat.Goto(ctx, idle)
		},
	)
}

func (sm *DBStateMachine) NewMachine() {
	var (
		idle = &State{StateType: DBIdle}
	)

	sm.StateMachine.New(idle)
	sm.SetInitialState(idle)
	sm.Reservations = make([]Reservation, 0)

	goat.OnEvent(sm, idle, &DBSelectEvent{},
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
		},
	)

	goat.OnEvent(sm, idle, &DBUpdateEvent{},
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
		},
	)

}

func (sm *ClientStateMachine) NewMachine(clientID int, roomID int, server *ServerStateMachine) {
	var (
		idle       = &State{StateType: ClientIdle}
		requesting = &State{StateType: ClientRequesting}
		end        = &State{StateType: ClientEnd}
	)

	sm.StateMachine.New(idle, requesting, end)
	sm.SetInitialState(idle)
	sm.ClientID = clientID
	sm.TargetRoom = roomID
	sm.Server = server

	goat.OnEntry(sm, idle, 
		func(ctx context.Context, client *ClientStateMachine) {
			requestEvent := &ReservationRequestEvent{
				RoomID:   client.TargetRoom,
				ClientID: client.ClientID,
				Client:   client,
			}

			goat.SendTo(ctx, client.Server, requestEvent)
			goat.Goto(ctx, requesting)
		},
	)

	goat.OnEvent(sm, requesting, &ReservationResultEvent{},
		func(ctx context.Context, event *ReservationResultEvent, client *ClientStateMachine) {
			if event.ClientID == client.ClientID {
				if event.Succeeded {
					goat.Goto(ctx, end)
				} else {
					goat.Goto(ctx, end)
				}
			}
		},
	)

	goat.OnEntry(sm, end,
		func(ctx context.Context, client *ClientStateMachine) {
		},
	)
}

func main() {
	// Create database
	db := &DBStateMachine{}
	db.NewMachine()

	// Create server
	server1 := &ServerStateMachine{}
	server1.NewMachine(db)

	// Create two clients trying to reserve the same room
	client1 := &ClientStateMachine{}
	client1.NewMachine(0, 101, server1) // Client 0, Room 101

	server2 := &ServerStateMachine{}
	server2.NewMachine(db)

	client2 := &ClientStateMachine{}
	client2.NewMachine(1, 101, server2) // Client 1, Same Room 101

	// Create Kripke model
	kripke, err := goat.KripkeModel(
		goat.WithStateMachines(server1, server2, db, client1, client2),
		goat.WithInvariants(
			// Invariant: A room should not be reserved by multiple clients
			goat.ToRef(db).Invariant(func(sm goat.AbstractStateMachine) bool {
				db := sm.(*DBStateMachine)

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
	)
	if err != nil {
		panic(err)
	}

	if err := kripke.Solve(); err != nil {
		panic(err)
	}


	fmt.Println("Meeting Room Reservation System (Without Proper Exclusion Control)")
	fmt.Println("Simulating: SELECT → UPDATE (without locking)")
	kripke.WriteAsLog(os.Stdout, "A room should not be reserved by multiple clients")
}
