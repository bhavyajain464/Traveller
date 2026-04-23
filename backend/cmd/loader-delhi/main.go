package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/gtfs"
)

func main() {
	// Load Delhi Metro data
	metroPath := flag.String("metro", "../DMRC_GTFS", "Path to DMRC GTFS data directory")
	// Load Delhi Bus data
	busPath := flag.String("bus", "../GTFS", "Path to Delhi Bus GTFS data directory")
	// Option to load only one dataset
	loadMetroOnly := flag.Bool("metro-only", false, "Load only metro data")
	loadBusOnly := flag.Bool("bus-only", false, "Load only bus data")
	flag.Parse()

	cfg := config.Load()

	// Initialize database
	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var mergedData *gtfs.GTFSData

	if *loadMetroOnly {
		// Load only metro data
		log.Printf("Loading Delhi Metro data from: %s", *metroPath)
		parser := gtfs.NewParser(*metroPath)
		mergedData, err = parser.Parse()
		if err != nil {
			log.Fatalf("Failed to parse Metro GTFS data: %v", err)
		}
	} else if *loadBusOnly {
		// Load only bus data
		log.Printf("Loading Delhi Bus data from: %s", *busPath)
		parser := gtfs.NewParser(*busPath)
		mergedData, err = parser.Parse()
		if err != nil {
			log.Fatalf("Failed to parse Bus GTFS data: %v", err)
		}
	} else {
		// Load and merge both datasets
		log.Println("Loading Delhi Metro and Bus data...")
		log.Printf("  Metro: %s", *metroPath)
		log.Printf("  Bus: %s", *busPath)

		var datasets []*gtfs.GTFSData

		// Load metro feed
		if _, err := os.Stat(*metroPath); !os.IsNotExist(err) {
			metroParser := gtfs.NewParser(*metroPath)
			metroData, err := metroParser.Parse()
			if err != nil {
				log.Fatalf("Failed to parse Metro GTFS data: %v", err)
			}
			datasets = append(datasets, metroData)
			log.Println("✓ Metro data loaded")
		} else {
			log.Printf("Warning: Metro path does not exist: %s", *metroPath)
		}

		// Load bus feed
		if _, err := os.Stat(*busPath); !os.IsNotExist(err) {
			busParser := gtfs.NewParser(*busPath)
			busData, err := busParser.Parse()
			if err != nil {
				log.Fatalf("Failed to parse Bus GTFS data: %v", err)
			}
			datasets = append(datasets, busData)
			log.Println("✓ Bus data loaded")
		} else {
			log.Printf("Warning: Bus path does not exist: %s", *busPath)
		}

		// Merge feeds
		log.Println("Merging feeds...")
		aggregator := gtfs.NewAggregator()
		mergedData = aggregator.Merge(datasets...)
		log.Println("✓ Feeds merged")
	}

	// Validate data
	log.Println("Validating GTFS data...")
	validator := gtfs.NewValidator()
	if errs := validator.Validate(mergedData); len(errs) > 0 {
		log.Printf("Validation warnings (%d):", len(errs))
		for _, err := range errs {
			log.Printf("  - %v", err)
		}
		log.Println("Continuing with data load...")
	}

	// Load data into database
	log.Println("Loading data into database...")
	// Extract the underlying *sql.DB from the database.DB wrapper
	sqlDB := db.DB // Access the embedded *sql.DB
	loader := gtfs.NewLoader(sqlDB)
	if err := loader.Load(mergedData); err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	log.Println("=" + string(filepath.Separator) + strings.Repeat("=", 60))
	log.Println("Delhi GTFS data loaded successfully!")
	log.Println("=" + string(filepath.Separator) + strings.Repeat("=", 60))
	log.Printf("  Agencies: %d", len(mergedData.Agencies))
	log.Printf("  Routes: %d", len(mergedData.Routes))
	log.Printf("  Stops: %d", len(mergedData.Stops))
	log.Printf("  Trips: %d", len(mergedData.Trips))
	log.Printf("  Stop Times: %d", len(mergedData.StopTimes))
	log.Printf("  Calendar: %d", len(mergedData.Calendar))
	log.Println("=" + string(filepath.Separator) + strings.Repeat("=", 60))
}

