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

func put(client iidy.RPCerClient, list string, item string) {
	r, err := client.Put(context.Background(), &iidy.Entry{List: list, Item: item})
	if err != nil {
		log.Fatalf("could not put: %v", err)
	}
	log.Printf("Successfully Put %v", r)
}

func get(client iidy.RPCerClient, list string, item string) {
	r, err := client.Get(context.Background(), &iidy.Entry{List: list, Item: item})
	if err != nil {
		log.Fatalf("could not get: %v", err)
	}
	log.Printf("Successfully Got %v", r)
	log.Printf("Attempts: %v", r.Attempts)
}

func main() {
	// Set up a connection to the server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := iidy.NewRPCerClient(conn)

	// Contact the server and print out its response.
	if len(os.Args) != 4 {
		log.Fatalf("Must provide a verb, a list, and an item name")
	}
	verb := os.Args[1]
	list := os.Args[2]
	item := os.Args[3]
	switch verb {
	case "put":
		put(client, list, item)
	case "get":
		get(client, list, item)
	default:
		log.Fatalf("do not know how to %s\n", verb)
	}
}
