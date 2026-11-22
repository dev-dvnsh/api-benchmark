package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RunRequest struct {
	URL           string            `json:"url"`           // Required
	Method        string            `json:"method"`        // default=GET
	Headers       map[string]string `json:"headers"`       // optional
	Body          json.RawMessage   `json:"body"`          // optional
	TotalRequests int               `json:"totalRequests"` // required
	Concurrency   int               `json:"concurrency"`   // required
}

func run(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "HELLO")
}

func main() {
	http.HandleFunc("/run", run)
	http.ListenAndServe(":8000", nil)
}
