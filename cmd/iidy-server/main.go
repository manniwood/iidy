package main

import (
	"context"
	"log"
	"net"

	"github.com/manniwood/iidy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// TODO: presumably a reference to an iidy pg store would go in the server
type server struct{}

// Put implements iidy.RPCerServer
func (s *server) Put(ctx context.Context, in *iidy.Entry) (*iidy.Reply, error) {
	return &iidy.Reply{Verb: "PUT", Count: 1}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	iidy.RegisterRPCerServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
