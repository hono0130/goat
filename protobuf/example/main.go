package main

import (
	"context"
	"fmt"
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

	// Example 1: Generate Protobuf specification
	fmt.Println("Example 1: Generating Protobuf Specification")
	fmt.Println("=============================================")
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
	fmt.Println("✓ Protobuf specification generated successfully")
	fmt.Println()

	// Example 2: Generate E2E test code
	fmt.Println("Example 2: Generating E2E Test Code")
	fmt.Println("====================================")
	err = protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		Spec:        spec,
		OutputDir:   "./testdata",
		PackageName: "main",
		Filename:    "user_service_e2e_test.go",
		TestCases: []protobuf.TestCase{
			{
				MethodName: "CreateUser",
				Inputs: []protobuf.AbstractProtobufMessage{
					&CreateUserRequest{Username: "alice", Email: "alice@example.com"},
				},
			},
			{
				MethodName: "GetUser",
				Inputs: []protobuf.AbstractProtobufMessage{
					&GetUserRequest{UserID: "user_123"},
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("GenerateE2ETest() error = %v", err)
	}
	fmt.Println("✓ E2E test code generated successfully")
	fmt.Println()
	fmt.Println("All examples completed successfully!")
}
