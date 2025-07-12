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

	sm.OnEvent(idle, &ReservationRequestEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*ReservationRequestEvent)
			sm.CurrentRequest = e

			// First, SELECT to check if room is already reserved
			selectEvent := &DBSelectEvent{
				RoomID:   e.RoomID,
				ClientID: e.ClientID,
				Server:   sm,
			}

			sm.SendUnary(sm.DB, selectEvent, env)
			sm.Goto(processing, env)
		},
	)

		// Handle SELECT result
		sm.OnEvent(processing, &DBSelectResultEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*DBSelectResultEvent)

			if sm.CurrentRequest == nil {
				sm.Goto(idle, env)
				return
			}

			if e.IsReserved {
				// Room is already reserved, send failure to client
				resultEvent := &ReservationResultEvent{
					RoomID:    sm.CurrentRequest.RoomID,
					ClientID:  sm.CurrentRequest.ClientID,
					Succeeded: false,
				}

				sm.SendUnary(sm.CurrentRequest.Client, resultEvent, env)
				sm.CurrentRequest = nil
				sm.Goto(idle, env)
				return
			}

			// Room is not reserved, proceed with UPDATE
			updateEvent := &DBUpdateEvent{
				RoomID:   sm.CurrentRequest.RoomID,
				ClientID: sm.CurrentRequest.ClientID,
				Server:   sm,
			}

			sm.SendUnary(sm.DB, updateEvent, env)
		},
	)

	sm.OnEvent(processing, &DBUpdateResultEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*DBUpdateResultEvent)

			if sm.CurrentRequest == nil {
				sm.Goto(idle, env)
				return
			}

			resultEvent := &ReservationResultEvent{
				RoomID:    sm.CurrentRequest.RoomID,
				ClientID:  sm.CurrentRequest.ClientID,
				Succeeded: e.Succeeded,
			}

			sm.SendUnary(sm.CurrentRequest.Client, resultEvent, env)
			sm.CurrentRequest = nil
			sm.Goto(idle, env)
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

	sm.OnEvent(idle, &DBSelectEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*DBSelectEvent)

			// Check if room is already reserved
			isReserved := false
			for _, res := range sm.Reservations {
				if res.RoomID == e.RoomID {
					isReserved = true
					break
				}
			}

			resultEvent := &DBSelectResultEvent{
				RoomID:     e.RoomID,
				ClientID:   e.ClientID,
				IsReserved: isReserved,
			}

			sm.SendUnary(e.Server, resultEvent, env)
		},
	)

	sm.OnEvent(idle, &DBUpdateEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*DBUpdateEvent)

			sm.Reservations = append(sm.Reservations, Reservation{
				UUID:     uuid.New().String(),
				RoomID:   e.RoomID,
				ClientID: e.ClientID,
			})

			resultEvent := &DBUpdateResultEvent{
				RoomID:    e.RoomID,
				ClientID:  e.ClientID,
				Succeeded: true, 
			}

			sm.SendUnary(e.Server, resultEvent, env)
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

	sm.OnEntry(idle,
		func(env *goat.Environment) {
			requestEvent := &ReservationRequestEvent{
				RoomID:   sm.TargetRoom,
				ClientID: sm.ClientID,
				Client:   sm,
			}

			sm.SendUnary(sm.Server, requestEvent, env)
			sm.Goto(requesting, env)
		},
	)

	sm.OnEvent(requesting, &ReservationResultEvent{},
		func(event goat.AbstractEvent, env *goat.Environment) {
			e := event.(*ReservationResultEvent)

			if e.ClientID == sm.ClientID {
				if e.Succeeded {
					sm.Goto(end, env)
				} else {
					sm.Goto(end, env)
				}
			}
		},
	)

	sm.OnEntry(end,
		func(env *goat.Environment) {
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
