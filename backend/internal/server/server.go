package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/plaid"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/snaptrade"
)

// Configures and starts the HTTP server with CORS support and protected routes.
func Run() error {
	mux := http.NewServeMux()

	// Initialize Supabase database client
	var dbClient *database.Client
	if client, err := database.NewClientFromEnv(); err != nil {
		log.Printf("supabase database client not configured: %v", err)
	} else {
		dbClient = client
		log.Println("supabase database client initialized")
	}

	// Initialize Plaid client
	var plaidClient *plaid.Client
	if client, err := plaid.NewClientFromEnv(); err != nil {
		log.Printf("plaid client not configured: %v", err)
	} else {
		plaidClient = client
	}

	// Initialize Snaptrade client
	var snaptradeClient *snaptrade.Client
	if client, err := snaptrade.NewClientFromEnv(); err != nil {
		log.Printf("snaptrade client not configured: %v", err)
	} else {
		snaptradeClient = client
	}

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

	// Registers the link management routes.
	registerLinkManagementRoutes(mux, apiDependencies{
		db:              dbClient,
		plaidClient:     plaidClient,
		snaptradeClient: snaptradeClient,
	})

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

// Adds CORS headers to responses, allowing the frontend origin to make requests.
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

// Retrieves an environment variable value, returning the fallback if not set.
func getEnv(key, fallback string) string {
	envValue := os.Getenv(key)
	if envValue != "" {
		return envValue
	}
	return fallback
}
