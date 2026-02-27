package handler

import (
	"log"
	"net/http"
	"sync"

	"github.com/matthewtzong/portfolio-tracker/backend/pkg/server"
)

var (
	handler http.Handler
	once    sync.Once
)

// Initializes the backend handler once and then serves the request.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		var err error
		handler, err = server.NewHandler()
		if err != nil {
			log.Printf("failed to initialize handler: %v", err)
		}
	})

	if handler == nil {
		http.Error(w, "internal server error: handler not initialized", http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
