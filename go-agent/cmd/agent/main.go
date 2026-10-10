package main

import (
	"encoding/json" // Converts Go structs into JSON responses
	"log"           // Writes application logs to the terminal
	"net/http"      // Go's built-in HTTP server and routing package
	"time"          // Provides timestamps and timeout functionality
)

// HealthResponse represents the JSON structure returned by our health API.
//
// The `json` tags define the property names in the JSON response.
// For example, the Go field "Status" becomes "status" in JSON.
//
// In Go, struct fields must start with an uppercase letter to be
// accessible by packages such as encoding/json.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// healthHandler processes incoming HTTP requests to /api/health.
//
// w (http.ResponseWriter):
//
//	Used to send HTTP status codes, headers, and response data.
//
// r (*http.Request):
//
//	Contains information about the incoming request,
//	including its HTTP method, headers, and URL.
func healthHandler(w http.ResponseWriter, r *http.Request) {

	// Only allow HTTP GET requests.
	// Return HTTP 405 for unsupported methods such as POST or DELETE.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a HealthResponse instance.
	// For now, we return a static healthy status.
	// Later, we'll add actual system and service health checks.
	response := HealthResponse{
		Status:  "healthy",
		Service: "go-monitoring-agent",

		// Generate the current UTC timestamp in ISO 8601-compatible
		// RFC3339 format, such as 2026-10-09T23:00:00Z.
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Tell the client that our response contains JSON data.
	w.Header().Set("Content-Type", "application/json")

	// Convert the Go struct into JSON and write it to the response.
	// Encode() automatically appends a newline to the JSON output.
	if err := json.NewEncoder(w).Encode(response); err != nil {

		// Log serialization or response-writing errors.
		// Avoid exposing internal error details to API clients.
		log.Printf("Error writing health response: %v", err)
	}
}

// main is the entry point of our Go application.
// It runs automatically when we execute the compiled program.
func main() {

	// Create a new HTTP request multiplexer (router).
	// This maps URL paths to their corresponding handler functions.
	mux := http.NewServeMux()

	// Register the health endpoint.
	// Requests to /api/health will execute healthHandler().
	mux.HandleFunc("/api/health", healthHandler)

	// Configure our HTTP server.
	server := &http.Server{

		// Bind to localhost only for security.
		// External computers cannot directly access this address.
		Addr: "127.0.0.1:9000",

		// Use our custom router to handle incoming requests.
		Handler: mux,

		// Limit the time allowed to read incoming HTTP headers.
		// This helps protect against slow-client resource exhaustion.
		ReadHeaderTimeout: 5 * time.Second,

		// Limit how long the server waits while reading requests.
		ReadTimeout: 10 * time.Second,

		// Limit the time allowed to write HTTP responses.
		WriteTimeout: 10 * time.Second,

		// Close idle keep-alive connections after 60 seconds.
		IdleTimeout: 60 * time.Second,
	}

	// Log the address where our API is available.
	log.Println("Go Monitoring Agent starting...")
	log.Println("Health API: http://127.0.0.1:9000/api/health")

	// Start listening for incoming HTTP requests.
	// ListenAndServe blocks while the server is running.
	//
	// If the server cannot start (for example, port 9000 is
	// already occupied), the error is logged and the app exits.
	log.Fatal(server.ListenAndServe())
}
