package main

import (
	"context"
	"log"
	"net"

	"github.com/manniwood/iidy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	Store *iidy.PgStore
}

// Put implements iidy.RPCerServer
func (s *server) Put(ctx context.Context, in *iidy.Entry) (*iidy.Reply, error) {
	count, err := s.Store.Add(in.List, in.Item)
	if err != nil {
		return nil, err
	}
	return &iidy.Reply{Verb: "ADDED", Count: count}, nil
}

// Get implements iidy.RPCerServer
func (s *server) Get(ctx context.Context, in *iidy.Entry) (*iidy.GetReply, error) {
	attempts, ok, err := s.Store.Get(in.List, in.Item)
	if err != nil {
		return nil, err
	}
	return &iidy.GetReply{Attempts: int64(attempts), Ok: ok}, nil
}

func main() {
	pgStore, err := iidy.NewPgStore()
	if err != nil {
		log.Fatalf("Problem with store: %v\n", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	iidy.RegisterRPCerServer(s, &server{Store: pgStore})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
