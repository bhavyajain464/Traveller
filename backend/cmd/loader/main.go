package main

import (
	"flag"
	"log"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/gtfs"
)

func main() {
	dataPath := flag.String("data", "", "Path to GTFS data directory")
	flag.Parse()

	if *dataPath == "" {
		cfg := config.Load()
		*dataPath = cfg.GTFS.DataPath
	}

	if *dataPath == "" {
		log.Fatal("GTFS data path is required. Use -data flag or set GTFS_DATA_PATH environment variable")
	}

	cfg := config.Load()

	// Initialize database
	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Parse GTFS data
	log.Printf("Parsing GTFS data from: %s", *dataPath)
	parser := gtfs.NewParser(*dataPath)
	data, err := parser.Parse()
	if err != nil {
		log.Fatalf("Failed to parse GTFS data: %v", err)
	}

	// Validate data
	log.Println("Validating GTFS data...")
	validator := gtfs.NewValidator()
	if errs := validator.Validate(data); len(errs) > 0 {
		log.Printf("Validation warnings (%d):", len(errs))
		for _, err := range errs {
			log.Printf("  - %v", err)
		}
		log.Println("Continuing with data load...")
	}

	// Load data into database
	log.Println("Loading data into database...")
	loader := gtfs.NewLoader(db.DB)
	if err := loader.Load(data); err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	log.Println("GTFS data loaded successfully!")
	log.Printf("  Agencies: %d", len(data.Agencies))
	log.Printf("  Routes: %d", len(data.Routes))
	log.Printf("  Stops: %d", len(data.Stops))
	log.Printf("  Trips: %d", len(data.Trips))
	log.Printf("  Stop Times: %d", len(data.StopTimes))
	log.Printf("  Calendar: %d", len(data.Calendar))
}
