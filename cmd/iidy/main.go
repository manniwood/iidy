package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/manniwood/iidy"
)

func main() {
	port := 8080

	s, err := iidy.NewPgStore()
	if err != nil {
		log.Fatalf("Problem with store: %v\n", err)
	}
	env := &iidy.Env{Store: s}

	http.Handle("/lists/", iidy.Handler{Env: env, H: iidy.ListHandler})

	log.Printf("Server starting on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
