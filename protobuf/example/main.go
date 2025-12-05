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
	protobuf.Message[*UserService, *UserService]
	Username string
	Email    string
	Tags     []string
}

type CreateUserResponse struct {
	protobuf.Message[*UserService, *UserService]
	UserID    string
	Success   bool
	ErrorCode int64
}

type GetUserRequest struct {
	protobuf.Message[*UserService, *UserService]
	UserID string
}

type GetUserResponse struct {
	protobuf.Message[*UserService, *UserService]
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

func createUserServiceModel() *protobuf.ServiceSpec[*UserService] {
	spec := protobuf.NewServiceSpec(&UserService{})
	idleState := &UserServiceState{StateType: UserServiceIdle}
	processingState := &UserServiceState{StateType: UserServiceProcessing}

	spec.DefineStates(idleState, processingState).SetInitialState(idleState)

	// Register RPC methods using OnMessage
	protobuf.OnMessage(spec, idleState, "CreateUser",
		func(ctx context.Context, event *CreateUserRequest, service *UserService) protobuf.Response[*CreateUserResponse] {
			response := &CreateUserResponse{
				UserID:    "user_123",
				Success:   true,
				ErrorCode: 0,
			}
			return protobuf.SendTo(ctx, service, response)
		})

	protobuf.OnMessage(spec, idleState, "GetUser",
		func(ctx context.Context, event *GetUserRequest, service *UserService) protobuf.Response[*GetUserResponse] {
			response := &GetUserResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Found:    true,
			}
			return protobuf.SendTo(ctx, service, response)
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

	err := protobuf.Generate(opts, spec)
	if err != nil {
		log.Fatalf("Generate() error = %v", err)
	}
	fmt.Println("✓ Protobuf specification generated successfully")
	fmt.Println()

	// Example 2: Generate E2E test code
	fmt.Println("Example 2: Generating E2E Test Code")
	fmt.Println("====================================")
	err = protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		OutputDir:   "./testdata",
		PackageName: "main",
		Services: []protobuf.ServiceTestCase{
			{
				Spec:           spec,
				ServicePackage: "github.com/goatx/goat/user/proto",
				Methods: []protobuf.MethodTestCase{
					{
						MethodName: "CreateUser",
						Inputs: []protobuf.AbstractMessage{
							&CreateUserRequest{Username: "alice", Email: "alice@example.com"},
							&CreateUserRequest{Username: "bob", Email: "bob@example.com"},
						},
					},
					{
						MethodName: "GetUser",
						Inputs: []protobuf.AbstractMessage{
							&GetUserRequest{UserID: "user_123"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("GenerateE2ETest() error = %v", err)
	}
	fmt.Println("✓ E2E test code generated successfully")
	fmt.Println("  Files generated:")
	fmt.Println("    - ./testdata/main_test.go")
	fmt.Println("    - ./testdata/user_service_test.go")
	fmt.Println()
	fmt.Println("All examples completed successfully!")
}
