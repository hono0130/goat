package main

import (
	"context"
	"testing"

	pbuser_service "github.com/goatx/goat/user/proto"
)

// TestCreateUser tests the CreateUser RPC call.
// This test was automatically generated from model checking execution.
func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		input    *pbuser_service.CreateUserRequest
		expected *pbuser_service.CreateUserResponse
	}{
		{
			name: "case_0",
			input: &pbuser_service.CreateUserRequest{
				Email:    "alice@example.com",
				Tags:     []string(nil),
				Username: "alice",
			},
			expected: &pbuser_service.CreateUserResponse{
				ErrorCode: 0,
				Success:   true,
				UserID:    "user_123",
			},
		},
		{
			name: "case_1",
			input: &pbuser_service.CreateUserRequest{
				Email:    "bob@example.com",
				Tags:     []string(nil),
				Username: "bob",
			},
			expected: &pbuser_service.CreateUserResponse{
				ErrorCode: 0,
				Success:   true,
				UserID:    "user_123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, err := user_serviceClient.CreateUser(ctx, tt.input)
			if err != nil {
				t.Fatalf("RPC call failed: %%v", err)
			}

			// Verify the output matches expected
			if !compareE2EOutput(tt.expected, actual) {
				t.Errorf("CreateUser output mismatch:\nexpected: %+v\ngot:      %+v", tt.expected, actual)
			}
		})
	}
}

// TestGetUser tests the GetUser RPC call.
// This test was automatically generated from model checking execution.
func TestGetUser(t *testing.T) {
	tests := []struct {
		name     string
		input    *pbuser_service.GetUserRequest
		expected *pbuser_service.GetUserResponse
	}{
		{
			name: "case_0",
			input: &pbuser_service.GetUserRequest{
				UserID: "user_123",
			},
			expected: &pbuser_service.GetUserResponse{
				Email:    "test@example.com",
				Found:    true,
				Username: "testuser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, err := user_serviceClient.GetUser(ctx, tt.input)
			if err != nil {
				t.Fatalf("RPC call failed: %%v", err)
			}

			// Verify the output matches expected
			if !compareE2EOutput(tt.expected, actual) {
				t.Errorf("GetUser output mismatch:\nexpected: %+v\ngot:      %+v", tt.expected, actual)
			}
		})
	}
}
