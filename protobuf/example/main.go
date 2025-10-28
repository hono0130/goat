package main

import (
	"context"
	"log"

	"github.com/goatx/goat"
	"github.com/goatx/goat/protobuf"
)

type UserService struct {
	goat.StateMachine
}

type CreateUserRequest struct {
	protobuf.ProtobufMessage[*UserService, *UserService]
	Username string
	Email    string
	Tags     []string
}

type CreateUserResponse struct {
	protobuf.ProtobufMessage[*UserService, *UserService]
	UserID    string
	Success   bool
	ErrorCode int64
}

type GetUserRequest struct {
	protobuf.ProtobufMessage[*UserService, *UserService]
	UserID string
}

type GetUserResponse struct {
	protobuf.ProtobufMessage[*UserService, *UserService]
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

func createUserServiceModel() *protobuf.ProtobufServiceSpec[*UserService] {
	spec := protobuf.NewProtobufServiceSpec(&UserService{})
	idleState := &UserServiceState{StateType: UserServiceIdle}
	processingState := &UserServiceState{StateType: UserServiceProcessing}

	spec.DefineStates(idleState, processingState).SetInitialState(idleState)

	// Register RPC methods using OnProtobufMessage
	protobuf.OnProtobufMessage(spec, idleState, "CreateUser",
		&CreateUserRequest{}, &CreateUserResponse{},
		func(ctx context.Context, event *CreateUserRequest, service *UserService) protobuf.ProtobufResponse[*CreateUserResponse] {
			response := &CreateUserResponse{
				UserID:    "user_123",
				Success:   true,
				ErrorCode: 0,
			}
			return protobuf.ProtobufSendTo(ctx, service, response)
		})

	protobuf.OnProtobufMessage(spec, idleState, "GetUser",
		&GetUserRequest{}, &GetUserResponse{},
		func(ctx context.Context, event *GetUserRequest, service *UserService) protobuf.ProtobufResponse[*GetUserResponse] {
			response := &GetUserResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Found:    true,
			}
			return protobuf.ProtobufSendTo(ctx, service, response)
		})

	return spec
}

func main() {
	spec := createUserServiceModel()

	opts := protobuf.GenerateOptions{
		OutputDir:   "./proto",
		PackageName: "user.service",
		GoPackage:   "github.com/goatx/goat/user/proto",
		Filename:    "user_service.proto",
	}

	err := protobuf.GenerateProtobuf(opts, spec)
	if err != nil {
		log.Fatalf("GenerateProtobuf() error = %v", err)
	}
}
