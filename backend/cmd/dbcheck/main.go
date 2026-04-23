package main

import (
	"fmt"
	"os"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
)

func main() {
	cfg := config.Load()
	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		fmt.Printf("Connection Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var count int
	
	err = db.DB.QueryRow("SELECT COUNT(*) FROM agencies").Scan(&count)
	if err != nil {
		fmt.Printf("Agencies error: %v\n", err)
	} else {
		fmt.Printf("Agencies: %d\n", count)
	}

	err = db.DB.QueryRow("SELECT COUNT(*) FROM routes").Scan(&count)
	if err != nil {
		fmt.Printf("Routes error: %v\n", err)
	} else {
		fmt.Printf("Routes: %d\n", count)
	}

	err = db.DB.QueryRow("SELECT COUNT(*) FROM stops").Scan(&count)
	if err != nil {
		fmt.Printf("Stops error: %v\n", err)
	} else {
		fmt.Printf("Stops: %d\n", count)
	}

	err = db.DB.QueryRow("SELECT COUNT(*) FROM trips").Scan(&count)
	if err != nil {
		fmt.Printf("Trips error: %v\n", err)
	} else {
		fmt.Printf("Trips: %d\n", count)
	}
}
