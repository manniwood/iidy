package iidy

import (
	"fmt"
	"net/http"
)

func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World\n")
}
