package main

import (
	"context"
	"log"
	"os"

	"github.com/manniwood/iidy"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := iidy.NewRPCerClient(conn)

	// Contact the server and print out its response.
	if len(os.Args) != 3 {
		log.Fatalf("Must provide a list and item name")
	}
	list := os.Args[1]
	item := os.Args[2]
	r, err := c.Put(context.Background(), &iidy.Entry{List: list, Item: item})
	if err != nil {
		log.Fatalf("could not put: %v", err)
	}
	log.Printf("Successfully Put %v", r)
}
