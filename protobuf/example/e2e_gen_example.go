package main

import (
	"fmt"
	"log"

	"github.com/goatx/goat/protobuf"
)

// This example demonstrates how to generate E2E test code.
// The expected output is automatically calculated by executing the registered handlers.

func generateE2ETestExample() {
	fmt.Println("Generating E2E Test Code")
	fmt.Println("========================")
	fmt.Println()

	// Create the service specification with handlers
	spec := createUserServiceModel()

	// Generate E2E test - only input events need to be specified
	// The output is automatically calculated by executing the handler
	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
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
		log.Fatalf("Failed to generate E2E test: %v", err)
	}

	fmt.Println("✓ E2E test code generated successfully")
	fmt.Println("  File: ./testdata/user_service_e2e_test.go")
	fmt.Println()
	fmt.Println("You can now run:")
	fmt.Println("  go test ./testdata/user_service_e2e_test.go")
}

