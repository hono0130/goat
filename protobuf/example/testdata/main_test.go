package main

import (
	"context"
	"log"
	"net"
	"os"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	pbuser_service "github.com/goatx/goat/user/proto"
)

var user_serviceClient pbuser_service.UserServiceClient

func TestMain(m *testing.M) {
	// Start UserService server
	lis0, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("Failed to listen: %%v", err)
	}

	grpcServer0 := grpc.NewServer()
	// TODO: Register your service implementation here
	// pbuser_service.RegisterUserServiceServer(grpcServer0, &yourServiceImplementation{})

	go func() {
		if err := grpcServer0.Serve(lis0); err != nil {
			log.Fatalf("Failed to serve: %%v", err)
		}
	}()

	// Create UserService client
	conn0, err := grpc.Dial(lis0.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial: %%v", err)
	}
	user_serviceClient = pbuser_service.NewUserServiceClient(conn0)

	// Run tests
	code := m.Run()

	// Cleanup
	conn0.Close()
	grpcServer0.Stop()

	os.Exit(code)
}

// compareE2EOutput compares two values for equality in E2E tests.
// This is a helper function automatically generated for E2E testing.
func compareE2EOutput(expected, actual any) bool {
	return reflect.DeepEqual(expected, actual)
}
