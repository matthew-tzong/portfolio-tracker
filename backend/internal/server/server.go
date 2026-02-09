package server

import (
	"log"
	"net/http"
	"os"
	"time"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
)

// Run configures and starts the HTTP server with CORS support and protected routes.
// It sets up a health check endpoint and protected API routes that require JWT authentication.
func Run() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Protected test endpoint
	mux.Handle("/api/protected/ping", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"pong (protected)"}`))
	})))

	handler := withCORS(mux)

	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("backend listening on :%s", port)
	return server.ListenAndServe()
}

// withCORS adds CORS headers to responses, allowing the frontend origin to make requests.
// It handles OPTIONS preflight requests and sets appropriate CORS headers for all other requests.
func withCORS(next http.Handler) http.Handler {
	allowedOrigin := getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// if the request is an OPTIONS preflight request, return 204 No Content
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}


// getEnv retrieves an environment variable value, returning the fallback if not set.
func getEnv(key, fallback string) string {
	envValue := os.Getenv(key); 
	if envValue != "" {
		return envValue
	}
	return fallback
}

