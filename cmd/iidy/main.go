package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/manniwood/iidy"
)

func main() {
	port := 8080

	http.HandleFunc("/helloworld", iidy.HelloWorldHandler)

	log.Printf("Server starting on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
