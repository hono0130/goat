package main

import (
	"context"
	"fmt"
	"log"

	"github.com/goatx/goat/protobuf"
)

// This example demonstrates how to generate E2E test code.

func generateE2ETestExample() {
	fmt.Println("Generating E2E Test Code")
	fmt.Println("========================")
	fmt.Println()

	// Generate E2E test
	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		OutputDir:   "./testdata",
		PackageName: "main",
		Filename:    "user_service_e2e_test.go",
		TestCases: []protobuf.TestCase{
			{
				MethodName: "CreateUser",
				Input:      &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
				GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
					// Execute handler to get output
					// In real scenario, you would call the actual handler
					return &CreateUserResponse{
						UserID:    "user_123",
						Success:   true,
						ErrorCode: 0,
					}, nil
				},
			},
			{
				MethodName: "GetUser",
				Input:      &GetUserRequest{UserID: "user_123"},
				GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
					// Execute handler to get output
					return &GetUserResponse{
						Username: "alice",
						Email:    "alice@example.com",
						Found:    true,
					}, nil
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

// Example showing how GetOutput would work with actual handler execution
func exampleWithRealHandler() {
	spec := createUserServiceModel()

	// You would extract the handler from the spec
	// and execute it here to get the actual output
	_ = spec

	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		OutputDir:   "./testdata",
		PackageName: "main",
		Filename:    "real_handler_test.go",
		TestCases: []protobuf.TestCase{
			{
				MethodName: "CreateUser",
				Input:      &CreateUserRequest{Username: "bob", Email: "bob@example.com"},
				GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
					// Here you would execute the actual handler from the spec
					// For now, we return a mock response
					ctx := context.Background()
					_ = ctx

					// In real implementation:
					// handler := spec.GetHandler("CreateUser")
					// response := handler.Execute(ctx, input)
					// return response, nil

					return &CreateUserResponse{
						UserID:    "user_456",
						Success:   true,
						ErrorCode: 0,
					}, nil
				},
			},
		},
	})

	if err != nil {
		log.Printf("Failed: %v", err)
	}
}
