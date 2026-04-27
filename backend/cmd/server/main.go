package main

import (
	"log"

	"indian-transit-backend/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("Warning: failed to close application resources: %v", err)
		}
	}()

	if err := application.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
