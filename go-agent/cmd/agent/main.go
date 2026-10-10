package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	// Import our internal system metrics package.
	"github.com/skyairwater/ubuntu-observability-dashboard/go-agent/internal/metrics"
)

// HealthResponse represents the basic service health response.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// healthHandler confirms that the Go monitoring API is running.
func healthHandler(w http.ResponseWriter, r *http.Request) {

	// Allow only GET requests.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Service:   "go-monitoring-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Tell the browser that the response contains JSON.
	w.Header().Set("Content-Type", "application/json")

	// Serialize our struct and send it to the client.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Health response error: %v", err)
	}
}

// systemHandler returns real CPU, RAM, disk, and host metrics.
func systemHandler(w http.ResponseWriter, r *http.Request) {

	// Only allow HTTP GET requests.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit metric collection to five seconds.
	// This prevents a slow system call from blocking forever.
	ctx, cancel := context.WithTimeout(
		r.Context(),
		5*time.Second,
	)
	defer cancel()

	// Call our dedicated metrics collection package.
	systemData, err := metrics.CollectSystemMetrics(ctx)

	if err != nil {
		// Log details internally for troubleshooting.
		log.Printf("System metrics collection failed: %v", err)

		// Return a generic error to the API consumer.
		http.Error(
			w,
			"Unable to collect system metrics",
			http.StatusInternalServerError,
		)
		return
	}

	// Set JSON content type before writing the response.
	w.Header().Set("Content-Type", "application/json")

	// Convert our collected metrics into JSON.
	if err := json.NewEncoder(w).Encode(systemData); err != nil {
		log.Printf("System response error: %v", err)
	}
}

// main initializes the HTTP server and registers API routes.
func main() {

	// Create our HTTP router.
	mux := http.NewServeMux()

	// Existing health-check endpoint.
	mux.HandleFunc("/api/health", healthHandler)

	// New system-monitoring endpoint.
	mux.HandleFunc("/api/system", systemHandler)

	// Configure the HTTP server.
	server := &http.Server{
		// Listen only on localhost.
		// The monitoring API is not publicly exposed.
		Addr: "127.0.0.1:9000",

		Handler: mux,

		// Protect against slow HTTP clients.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Go Monitoring Agent started")
	log.Println("Health: http://127.0.0.1:9000/api/health")
	log.Println("System: http://127.0.0.1:9000/api/system")

	// Start the HTTP server.
	// LogFatal terminates the application if the server fails.
	log.Fatal(server.ListenAndServe())
}
