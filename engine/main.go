package main

import (
	"encoding/json"
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

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method Not Allowed, Use POST"})
		return
	}
	defer r.Body.Close()
	var req RunRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Bad Request"})
		return
	}

	if req.Method == "" {
		req.Method = "GET"
	}
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Url should not be empty"})
		return
	}

	if req.Method != "GET" && req.Method != "POST" && req.Method != "PUT" && req.Method != "DELETE" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Method must be only GET/POST/PUT/DEETE"})
		return
	}

	if req.TotalRequests < 1 || req.TotalRequests > 500 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Too less or many requests"})
		return
	}

	if req.Concurrency < 1 || req.Concurrency > 50 || req.Concurrency > req.TotalRequests {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Concurrency too high or low"})
		return
	}

	writeJSON(w, http.StatusOK, req)
}

func main() {
	http.HandleFunc("/run", runHandler)
	http.ListenAndServe(":8000", nil)
}
