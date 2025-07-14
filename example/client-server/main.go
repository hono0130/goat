package main

import (
	"context"

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
	c.StateMachine.New(idle, waiting)
	c.SetInitialState(idle)

	// Define the default handlers for the idle state using new type-safe API
	goat.OnEntry(c, idle, func(ctx context.Context, client *Client) {
		reqCtx := Context{RequestID: randomRequestID()}
		goat.SendTo(ctx, client.Server, &eCheckMenuExistenceRequest{
			From:   client,
			Ctx:    reqCtx,
			MenuID: "menu_id",
		})
		goat.Goto(ctx, waiting)
	})
	
	goat.OnEntry(c, idle, func(ctx context.Context, client *Client) {
		client.Mut = 100000
		goat.Goto(ctx, waiting)
	})

	// Define the default handlers for the waiting state using new type-safe API
	goat.OnEvent(c, waiting, &eCheckMenuExistenceResponse{}, 
		func(ctx context.Context, event *eCheckMenuExistenceResponse, client *Client) {
			if event.Err {
				goat.Goto(ctx, idle)
				return
			}
		},
	)

}

func (s *Server) NewMachine() {
	var (
		init    = &ServerState{ServerState: ServerStateInit}
		running = &ServerState{ServerState: ServerStateRunning}
	)

	// [MUST] This is a must to call New() method to initialize the state machine.
	// [MUST] This is a must to call SetInitialState() method to set the initial state.
	s.StateMachine.New(init, running)
	s.SetInitialState(init)

	goat.OnEntry(s, init, func(ctx context.Context, server *Server) {
		goat.Goto(ctx, running)
	})

	goat.OnEvent(s, running, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: true,
			})
		},
	)
	
	goat.OnEvent(s, running, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: false,
				Err:    false,
			})
		},
	)
	
	goat.OnEvent(s, running, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: false,
				Err:    true,
			})
		},
	)
}

func main() {
	server := &Server{}
	server.NewMachine()

	client := &Client{}
	client.NewMachine(server)

	err := goat.Test(
		goat.WithStateMachines(server, client),
	)
	if err != nil {
		panic(err)
	}
}

func randomRequestID() string {
	return "random_request_id"
}
