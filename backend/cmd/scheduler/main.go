package main

import (
	"log"
	"time"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/repository"
	"indian-transit-backend/internal/services"
)

// Scheduler runs daily tasks like bill generation
func main() {
	cfg := config.Load()

	// Initialize database
	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	dailyBillRepo := repository.NewDailyBillRepository(db)
	journeySessionRepo := repository.NewJourneySessionRepository(db)
	fareTransactionRepo := repository.NewFareTransactionRepository(db)
	billService := services.NewDailyBillService(dailyBillRepo, journeySessionRepo, fareTransactionRepo)

	// Run bill generation daily at 1 AM
	ticker := time.NewTicker(24 * time.Hour)

	// Run immediately on startup for yesterday's bills
	yesterday := time.Now().AddDate(0, 0, -1)
	log.Printf("Generating bills for %s", yesterday.Format("2006-01-02"))
	if err := billService.GenerateDailyBills(yesterday); err != nil {
		log.Printf("Error generating bills: %v", err)
	} else {
		log.Printf("Successfully generated bills for %s", yesterday.Format("2006-01-02"))
	}

	// Schedule daily runs
	for range ticker.C {
		yesterday := time.Now().AddDate(0, 0, -1)
		log.Printf("Generating bills for %s", yesterday.Format("2006-01-02"))
		if err := billService.GenerateDailyBills(yesterday); err != nil {
			log.Printf("Error generating bills: %v", err)
		} else {
			log.Printf("Successfully generated bills for %s", yesterday.Format("2006-01-02"))
		}
	}
}
