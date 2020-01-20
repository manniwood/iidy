package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/manniwood/iidy"
)

func main() {
	port := 8080

	s, err := iidy.NewPgStore(os.Getenv("IIDY_PG_CONN_URL"))
	if err != nil {
		log.Fatalf("Could not connect to data store: %v\n", err)
	}
	log.Printf("Connecting to data store with following config:\n%s\n", s)
	h := &iidy.Handler{Store: s}

	http.Handle("/", h)

	log.Printf("Server starting on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
