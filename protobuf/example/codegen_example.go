package main

import (
	"fmt"
	"log"
	"os"

	"github.com/goatx/goat/protobuf"
)

// This example demonstrates how to generate Go test code from recorded RPC traces.

func generateE2ETestExample() {
	fmt.Println("Generating Go E2E Test Code")
	fmt.Println("============================")
	fmt.Println()

	// Create a test recorder
	recorder := protobuf.NewE2ETestRecorder(
		"user_service_e2e",
		"E2E tests for user service operations",
	)

	// Register event types
	recorder.RegisterEventType(&CreateUserRequest{})
	recorder.RegisterEventType(&CreateUserResponse{})
	recorder.RegisterEventType(&GetUserRequest{})
	recorder.RegisterEventType(&GetUserResponse{})

	// Create service instance
	userService := &UserService{}

	// Record some RPC calls
	fmt.Println("Recording RPC calls...")

	// CreateUser RPC
	recorder.Record("CreateUser", userService, userService,
		&CreateUserRequest{
			Username: "alice",
			Email:    "alice@example.com",
			Tags:     []string{"admin", "developer"},
		},
		&CreateUserResponse{
			UserID:    "user_123",
			Success:   true,
			ErrorCode: 0,
		},
		1,
	)
	fmt.Println("✓ Recorded CreateUser RPC")

	// GetUser RPC
	recorder.Record("GetUser", userService, userService,
		&GetUserRequest{
			UserID: "user_123",
		},
		&GetUserResponse{
			Username: "alice",
			Email:    "alice@example.com",
			Found:    true,
		},
		2,
	)
	fmt.Println("✓ Recorded GetUser RPC")

	fmt.Println()

	// Generate Go test code
	fmt.Println("Generating Go test code...")
	code, err := recorder.GenerateGoTest("main")
	if err != nil {
		log.Fatalf("Failed to generate test code: %v", err)
	}

	fmt.Println("✓ Test code generated successfully")
	fmt.Println()

	// Display the generated code
	fmt.Println("Generated Test Code:")
	fmt.Println("====================")
	fmt.Println(code)
	fmt.Println()

	// Optionally save to file
	testDir := "testdata"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Printf("Warning: Could not create test directory: %v", err)
		return
	}

	testFile := testDir + "/user_service_generated_test.go"
	err = recorder.GenerateGoTestToFile("main", testFile)
	if err != nil {
		log.Printf("Warning: Could not save test file: %v", err)
		return
	}

	fmt.Printf("✓ Test code saved to: %s\n", testFile)
	fmt.Println()
	fmt.Println("You can now run the generated tests with:")
	fmt.Println("  go test ./testdata/user_service_generated_test.go")
}

// quickCodeGenExample demonstrates the QuickRecord feature with code generation.
func quickCodeGenExample() {
	fmt.Println("Quick Code Generation Example")
	fmt.Println("==============================")
	fmt.Println()

	userService := &UserService{}

	// Quickly record a single RPC
	testCase := protobuf.QuickRecord(
		"quick_user_test",
		"CreateUser",
		userService,
		&CreateUserRequest{
			Username: "bob",
			Email:    "bob@example.com",
		},
		&CreateUserResponse{
			UserID:  "user_456",
			Success: true,
		},
	)

	// Generate test code
	generator := protobuf.NewGoTestGenerator("main")
	code, err := generator.Generate(testCase)
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	fmt.Println("Generated code:")
	fmt.Println(code)
}
