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

type CreateUserRequest struct {
	openapi.OpenAPIEndpoint[*UserService, *UserService]
	Username string
	Email    string
	Tags     []string
}

type CreateUserResponse struct {
	openapi.OpenAPIEndpoint[*UserService, *UserService]
	UserID    string
	Success   bool
	ErrorCode int64
}

type GetUserRequest struct {
	openapi.OpenAPIEndpoint[*UserService, *UserService]
	UserID string
}

type GetUserResponse struct {
	openapi.OpenAPIEndpoint[*UserService, *UserService]
	Username string
	Email    string
	Found    bool
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

func createUserServiceModel() *openapi.OpenAPIServiceSpec[*UserService] {
	spec := openapi.NewOpenAPIServiceSpec(&UserService{})
	idleState := &UserServiceState{StateType: UserServiceIdle}
	processingState := &UserServiceState{StateType: UserServiceProcessing}

	spec.DefineStates(idleState, processingState).SetInitialState(idleState)

	// Register API endpoints using OnOpenAPIEndpoint
	openapi.OnOpenAPIEndpoint[*UserService, *CreateUserRequest, *CreateUserResponse](spec, idleState, "POST", "/users", "createUser",
		func(ctx context.Context, event *CreateUserRequest, service *UserService) openapi.OpenAPIResponse[*CreateUserResponse] {
			response := &CreateUserResponse{
				UserID:    "user_123",
				Success:   true,
				ErrorCode: 0,
			}
			return openapi.OpenAPISendTo(ctx, service, response)
		})

	openapi.OnOpenAPIEndpoint[*UserService, *GetUserRequest, *GetUserResponse](spec, idleState, "GET", "/users/{userId}", "getUser",
		func(ctx context.Context, event *GetUserRequest, service *UserService) openapi.OpenAPIResponse[*GetUserResponse] {
			response := &GetUserResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Found:    true,
			}
			return openapi.OpenAPISendTo(ctx, service, response)
		})

	return spec
}

func main() {
	spec := createUserServiceModel()

	opts := openapi.GenerateOptions{
		OutputDir:   "./openapi-spec",
		Title:       "User Service API",
		Version:     "1.0.0",
		Description: "API for managing users",
		Filename:    "user_service.yaml",
	}

	err := openapi.GenerateOpenAPI(&opts, spec)
	if err != nil {
		log.Fatalf("GenerateOpenAPI() error = %v", err)
	}
}
