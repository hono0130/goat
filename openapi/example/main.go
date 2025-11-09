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
	openapi.OpenAPISchema[*UserService, *UserService]
	Username string
	Email    string
	Tags     []string
}

type CreateUserResponse struct {
	openapi.OpenAPISchema[*UserService, *UserService]
	UserID    string
	Success   bool
	ErrorCode int64
}

type GetUserRequest struct {
	openapi.OpenAPISchema[*UserService, *UserService]
	UserID string
}

type GetUserResponse struct {
	openapi.OpenAPISchema[*UserService, *UserService]
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

	// Register API endpoints using OnOpenAPIRequest
	openapi.OnOpenAPIRequest[*UserService, *CreateUserRequest, *CreateUserResponse](spec, idleState, "POST", "/users",
		func(ctx context.Context, event *CreateUserRequest, service *UserService) openapi.OpenAPIResponseWrapper[*CreateUserResponse] {
			response := &CreateUserResponse{
				UserID:    "user_123",
				Success:   true,
				ErrorCode: 0,
			}
			return openapi.OpenAPISendTo(ctx, service, response, openapi.StatusCreated)
		},
		openapi.WithOperationID("createUser"),
		openapi.WithStatusCode(openapi.StatusCreated))

	openapi.OnOpenAPIRequest[*UserService, *GetUserRequest, *GetUserResponse](spec, idleState, "GET", "/users/{userId}",
		func(ctx context.Context, event *GetUserRequest, service *UserService) openapi.OpenAPIResponseWrapper[*GetUserResponse] {
			response := &GetUserResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Found:    true,
			}
			return openapi.OpenAPISendTo(ctx, service, response, openapi.StatusOK)
		},
		openapi.WithOperationID("getUser"))

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
