package main

import (
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
	IsLocked   bool
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

// Server state machine
type ServerStateMachine struct {
	goat.StateMachine
	CurrentRequest *ReservationRequestEvent
	DB             *DBStateMachine
}

type DBStateMachine struct {
	goat.StateMachine
	Reservations []Reservation
	LockedRooms  map[int]int // Map of room ID to client ID that locked it
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

	sm.StateMachine.New()
	sm.SetInitialState(idle)
	sm.DB = db

	sm.WithState(idle,
		goat.WithOnEvent(&ReservationRequestEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			server := sm.(*ServerStateMachine)
			e := event.(*ReservationRequestEvent)
			server.CurrentRequest = e

			// First, SELECT FOR UPDATE to check and lock the room
			selectEvent := &DBSelectEvent{
				RoomID:   e.RoomID,
				ClientID: e.ClientID,
				Server:   server,
			}

			server.SendUnary(server.DB, selectEvent, env)
			server.Goto(processing, env)
		}),
	)

	sm.WithState(processing,
		// Handle SELECT result
		goat.WithOnEvent(&DBSelectResultEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			server := sm.(*ServerStateMachine)
			e := event.(*DBSelectResultEvent)

			if server.CurrentRequest == nil {
				server.Goto(idle, env)
				return
			}

			// Only proceed if we got the lock
			if e.IsLocked {
				// Only update if room is not already reserved
				if !e.IsReserved {
					updateEvent := &DBUpdateEvent{
						RoomID:   server.CurrentRequest.RoomID,
						ClientID: server.CurrentRequest.ClientID,
						Server:   server,
					}

					server.SendUnary(server.DB, updateEvent, env)
				} else {
					// Room is already reserved, send failure to client
					resultEvent := &ReservationResultEvent{
						RoomID:    server.CurrentRequest.RoomID,
						ClientID:  server.CurrentRequest.ClientID,
						Succeeded: false,
					}

					server.SendUnary(server.CurrentRequest.Client, resultEvent, env)
					server.CurrentRequest = nil
					server.Goto(idle, env)
				}
			} else {
				// Failed to get lock, send failure to client
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: false,
				}

				server.SendUnary(server.CurrentRequest.Client, resultEvent, env)
				server.CurrentRequest = nil
				server.Goto(idle, env)
			}
		}),

		goat.WithOnEvent(&DBUpdateResultEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			server := sm.(*ServerStateMachine)
			e := event.(*DBUpdateResultEvent)

			if server.CurrentRequest == nil {
				server.Goto(idle, env)
				return
			}

			resultEvent := &ReservationResultEvent{
				RoomID:    server.CurrentRequest.RoomID,
				ClientID:  server.CurrentRequest.ClientID,
				Succeeded: e.Succeeded,
			}

			server.SendUnary(server.CurrentRequest.Client, resultEvent, env)
			server.CurrentRequest = nil
			server.Goto(idle, env)
		}),
	)
}

func (sm *DBStateMachine) NewMachine() {
	var (
		idle = &State{StateType: DBIdle}
	)

	sm.StateMachine.New()
	sm.SetInitialState(idle)
	sm.Reservations = make([]Reservation, 0)
	sm.LockedRooms = make(map[int]int)

	sm.WithState(idle,
		goat.WithOnEvent(&DBSelectEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			db := sm.(*DBStateMachine)
			e := event.(*DBSelectEvent)

			// Check if room is already reserved
			isReserved := false
			for _, res := range db.Reservations {
				if res.RoomID == e.RoomID {
					isReserved = true
					break
				}
			}

			// Try to acquire lock on the room
			isLocked := false
			if _, exists := db.LockedRooms[e.RoomID]; !exists {
				// Room is not locked, acquire lock
				db.LockedRooms[e.RoomID] = e.ClientID
				isLocked = true
			}

			resultEvent := &DBSelectResultEvent{
				RoomID:     e.RoomID,
				ClientID:   e.ClientID,
				IsReserved: isReserved,
				IsLocked:   isLocked,
			}

			db.SendUnary(e.Server, resultEvent, env)
			db.Goto(idle, env)
		}),

		goat.WithOnEvent(&DBUpdateEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			db := sm.(*DBStateMachine)
			e := event.(*DBUpdateEvent)

			// Check if this client has the lock
			hasLock := false
			if clientID, exists := db.LockedRooms[e.RoomID]; exists && clientID == e.ClientID {
				hasLock = true
			}

			succeeded := false
			if hasLock {
				// Room is available, make reservation
				db.Reservations = append(db.Reservations, Reservation{
					UUID:     uuid.New().String(),
					RoomID:   e.RoomID,
					ClientID: e.ClientID,
				})
				succeeded = true
			}

			resultEvent := &DBUpdateResultEvent{
				RoomID:    e.RoomID,
				ClientID:  e.ClientID,
				Succeeded: succeeded,
			}

			db.SendUnary(e.Server, resultEvent, env)
			db.Goto(idle, env)
		}),
	)

}

func (sm *ClientStateMachine) NewMachine(clientID int, roomID int, server *ServerStateMachine) {
	var (
		idle       = &State{StateType: ClientIdle}
		requesting = &State{StateType: ClientRequesting}
		end        = &State{StateType: ClientEnd}
	)

	sm.StateMachine.New()
	sm.SetInitialState(idle)
	sm.ClientID = clientID
	sm.TargetRoom = roomID
	sm.Server = server

	sm.WithState(idle,
		goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {
			client := sm.(*ClientStateMachine)

			requestEvent := &ReservationRequestEvent{
				RoomID:   client.TargetRoom,
				ClientID: client.ClientID,
				Client:   client,
			}

			client.SendUnary(client.Server, requestEvent, env)
			client.Goto(requesting, env)
		}),
	)

	sm.WithState(requesting,
		goat.WithOnEvent(&ReservationResultEvent{}, func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
			client := sm.(*ClientStateMachine)
			e := event.(*ReservationResultEvent)

			if e.ClientID == client.ClientID {
				if e.Succeeded {
					client.Goto(end, env)
				} else {
					client.Goto(end, env)
				}
			}
		}),
	)

	sm.WithState(end, goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {}))
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

	fmt.Println("Meeting Room Reservation System (With Proper Exclusion Control)")
	fmt.Println("Simulating: SELECT FOR UPDATE → UPDATE (with locking)")
	kripke.WriteAsLog(os.Stdout, "A room should not be reserved by multiple clients")
}
