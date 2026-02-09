package main

import (
	"log"
	"os"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/server"
)

func main() {
	if err := server.Run() {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
	log.Println("server started")
}
