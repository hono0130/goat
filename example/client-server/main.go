package main

import (
	"os"

	"github.com/goatx/goat"
)

// User Defined Types for goat

type (
	// Context is a user-defined type for the context of the event.
	Context struct {
		RequestID string
	}
)

// Event Definitions

type (
	// eCheckMenuExistenceRequest is an event for checking the existence of the menu.
	// This event is sent from the client to the server.
	eCheckMenuExistenceRequest struct {
		goat.Event // [MUST] Embed the Event struct.
		Ctx        Context
		MenuID     string
		From       *Client
	}
	// eCheckMenuExistenceResponse is an event for the response of checking the existence of the menu.
	// This event is sent from the server to the client.
	eCheckMenuExistenceResponse struct {
		goat.Event // [MUST] Embed the Event struct.
		Exists     bool
		Err        bool
	}
)

// StateMachine State Definitions

type (
	// ClientStateType represents possible states of a Client.
	ClientStateType string
	// ServerStateType represents possible states of a Server.
	ServerStateType string
)

// Client states
const (
	ClientStateIdle    ClientStateType = "idle"    // The client is idle
	ClientStateWaiting                 = "waiting" // The client is waiting for a response
)

// Server states
const (
	ServerStateInit    ServerStateType = "init"    // The server is initializing
	ServerStateRunning                 = "running" // The server is running
)

type (
	// ClientState represents the state of a Client.
	ClientState struct {
		goat.State                  // [MUST] Embed the State struct.
		ClientState ClientStateType // The current state of the client
	}

	// ServerState represents the state of a Server.
	ServerState struct {
		goat.State                  // [MUST] Embed the State struct.
		ServerState ServerStateType // The current state of the server
	}
)

// StateMachine Definition

type (
	Client struct {
		goat.StateMachine // [MUST] Embed the StateMachine struct.
		// Mut is example of mutable field
		Mut int
		// server is the server that the client is connected to
		Server *Server
	}

	Server struct {
		goat.StateMachine // [MUST] Embed the StateMachine struct.
	}
)

func (c *Client) NewMachine(server *Server) {
	var (
		idle    = &ClientState{ClientState: ClientStateIdle}
		waiting = &ClientState{ClientState: ClientStateWaiting}
	)

	// Initialize the client.
	// Set the server that the client is connected to.
	c.Server = server
	// [MUST] This is a must to call New() method to initialize the state machine.
	// [MUST] This is a must to call SetInitialState() method to set the initial state.
	c.StateMachine.New()
	c.SetInitialState(idle)

	// Define the state machine.
	c.WithState(idle,
		goat.WithOnEntry(
			func(sm goat.AbstractStateMachine, env *goat.Environment) {
				this := sm.(*Client)
				ctx := Context{RequestID: randomRequestID()}
				this.SendUnary(c.Server, &eCheckMenuExistenceRequest{
					From:   c,
					Ctx:    ctx,
					MenuID: "menu_id",
				}, env)
				this.Goto(waiting, env)
			},
			func(sm goat.AbstractStateMachine, env *goat.Environment) {
				this := sm.(*Client)
				this.Mut = 100000
				this.Goto(waiting, env)
			},
		),
	)

	c.WithState(waiting,
		goat.WithOnEvent(&eCheckMenuExistenceResponse{},
			func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
				this := sm.(*Client)
				e := event.(*eCheckMenuExistenceResponse)
				if e.Err {
					this.Goto(idle, env)
					return
				}
			},
		),
	)
}

func (s *Server) NewMachine() {
	var (
		init    = &ServerState{ServerState: ServerStateInit}
		running = &ServerState{ServerState: ServerStateRunning}
	)

	// [MUST] This is a must to call New() method to initialize the state machine.
	// [MUST] This is a must to call SetInitialState() method to set the initial state.
	s.StateMachine.New()
	s.SetInitialState(init)

	s.WithState(init,
		goat.WithOnEntry(func(sm goat.AbstractStateMachine, env *goat.Environment) {
			this := sm.(*Server)
			this.Goto(running, env)
		}),
	)

	s.WithState(running,
		goat.WithOnEvent(&eCheckMenuExistenceRequest{},
			func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
				this := sm.(*Server)
				e := event.(*eCheckMenuExistenceRequest)
				this.SendUnary(e.From, &eCheckMenuExistenceResponse{
					Exists: true,
				}, env)
			},
			func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
				this := sm.(*Server)
				e := event.(*eCheckMenuExistenceRequest)
				this.SendUnary(e.From, &eCheckMenuExistenceResponse{
					Exists: false,
					Err:    false,
				}, env)
			},
			func(sm goat.AbstractStateMachine, event goat.AbstractEvent, env *goat.Environment) {
				this := sm.(*Server)
				e := event.(*eCheckMenuExistenceRequest)
				this.SendUnary(e.From, &eCheckMenuExistenceResponse{
					Exists: false,
					Err:    true,
				}, env)
			},
		),
	)
}

func main() {
	server := &Server{}
	server.NewMachine()

	client := &Client{}
	client.NewMachine(server)

	kripke, err := goat.KripkeModel(
		goat.WithStateMachines(server, client),
	)
	if err != nil {
		panic(err)
	}
	if err := kripke.Solve(); err != nil {
		panic(err)
	}
	kripke.WriteAsDot(os.Stdout)
}

func randomRequestID() string {
	return "random_request_id"
}
