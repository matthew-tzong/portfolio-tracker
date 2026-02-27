package main

import (
	"github.com/matthewtzong/portfolio-tracker/backend/pkg/server"
	"log"
	"os"
)

func main() {
	err := server.Run()
	if err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
