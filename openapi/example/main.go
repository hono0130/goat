package main

import (
	"context"
	"log"

	"github.com/goatx/goat"
	"github.com/goatx/goat/openapi"
)

type UserService struct {
	goat.StateMachine
}

type Cliennt struct {
	goat.StateMachine
}

type CreateUserRequest struct {
	openapi.Schema[*Cliennt, *UserService]
	Username string   `openapi:"required"`
	Email    string   `openapi:"required"`
	Tags     []string `openapi:"required"`
}

type CreateUserResponse struct {
	openapi.Schema[*UserService, *Cliennt]
	UserID  string `openapi:"required"`
	Success bool   `openapi:"required"`
}

type GetUserRequest struct {
	openapi.Schema[*Cliennt, *UserService]
	UserID       string `openapi:"path=userId"`
	IncludeEmail bool   `openapi:"query=includeEmail"`
	RequestID    string `openapi:"header=X-Request-ID,required"`
}

type GetUserResponse struct {
	openapi.Schema[*UserService, *Cliennt]
	Username string `openapi:"required"`
	Email    string `openapi:"required"`
	Found    bool   `openapi:"required"`
}

type GetUserNotFoundResponse struct {
	openapi.Schema[*UserService, *Cliennt]
	Message string `openapi:"required"`
}

type StateType string

const (
	UserServiceIdle       StateType = "idle"
	UserServiceProcessing StateType = "processing"
)

type UserServiceState struct {
	goat.State
	StateType StateType
}

func createUserServiceModel() *openapi.ServiceSpec[*UserService] {
	spec := openapi.NewServiceSpec(&UserService{})
	idleState := &UserServiceState{StateType: UserServiceIdle}
	processingState := &UserServiceState{StateType: UserServiceProcessing}

	spec.DefineStates(idleState, processingState).SetInitialState(idleState)

	openapi.OnRequest(spec, idleState, openapi.HTTPMethodPost, "/users",
		func(ctx context.Context, event *CreateUserRequest, service *UserService) openapi.Response[*CreateUserResponse] {
			client := event.Sender()
			response := &CreateUserResponse{
				UserID:  "user_123",
				Success: true,
			}
			return openapi.SendTo(ctx, client, response)
		},
		openapi.WithOperationID("createUser"),
		openapi.WithStatusCode(openapi.StatusCreated),
		openapi.WithRequestBodyOptional())

	openapi.OnRequest(spec, idleState, openapi.HTTPMethodGet, "/users/{userId}",
		func(ctx context.Context, event *GetUserRequest, service *UserService) openapi.Response[*GetUserResponse] {
			client := event.Sender()
			response := &GetUserResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Found:    true,
			}
			return openapi.SendTo(ctx, client, response)
		},
		openapi.WithOperationID("getUser"))

	openapi.OnRequest(spec, idleState, openapi.HTTPMethodGet, "/users/{userId}",
		func(ctx context.Context, event *GetUserRequest, service *UserService) openapi.Response[*GetUserNotFoundResponse] {
			client := event.Sender()
			response := &GetUserNotFoundResponse{
				Message: "user not found",
			}
			return openapi.SendTo(ctx, client, response)
		},
		openapi.WithOperationID("getUser"),
		openapi.WithStatusCode(openapi.StatusNotFound))

	return spec
}

func main() {
	spec := createUserServiceModel()

	opts := openapi.GenerateOptions{
		OutputDir: "./openapi",
		Title:     "User Service API",
		Version:   "1.0.0",
		Filename:  "user_service.yaml",
	}

	err := openapi.Generate(&opts, spec)
	if err != nil {
		log.Fatalf("Generate() error = %v", err)
	}
}
