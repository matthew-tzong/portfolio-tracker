package main

import (
	"log"
	"os"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/server"
)

func main() {
	err := server.Run()
	if err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
