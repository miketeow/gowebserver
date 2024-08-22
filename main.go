package main

import (
	"fmt"
	"net/http"
	"sync"
)

type apiConfig struct {
	fileserverHits int
	mu sync.Mutex
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter in a thread-safe manner
		cfg.mu.Lock()
		cfg.fileserverHits++
		cfg.mu.Unlock()

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the mutex to safely read fileserverHits
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	//Write the number of hits as plain text
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	htmlResponse := fmt.Sprintf(`
	<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>
	`, cfg.fileserverHits)

	w.Write([]byte(htmlResponse))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the mutex to safely reset fileserverHits
	cfg.mu.Lock()
	cfg.fileserverHits = 0
	cfg.mu.Unlock()

	//Respond with a 200 OK
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits counter reset"))
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	//Write the content type header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	//Write the status code 200 OK
	w.WriteHeader(http.StatusOK)
	//Write the body text OK
	w.Write([]byte("OK"))
}

func main() {

	// Initialize the API configuration
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	//Readiness endpoint at /healthz, restrict to GET method only
	mux.HandleFunc("GET /api/healthz", healthzHandler)

	//File server to serve files from the current directory under /app/*
	fileServer := http.FileServer(http.Dir("."))
	// Wrap the file server with middleware
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	//Metrics endpoint at /metrics, restrict to GET method only
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)

	mux.HandleFunc("/api/reset",apiCfg.resetHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux, // Use the ServeMux as the service handler
	}

	fmt.Println("Starting server on port 8080")
	server.ListenAndServe()
}