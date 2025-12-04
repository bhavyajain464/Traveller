package main

import (
	"flag"
	"log"
	"os"

	"data-scraper/internal/irctc"
)

func main() {
	outputDir := flag.String("output", "output/irctc", "Output directory for GTFS files")
	flag.Parse()

	scraper := irctc.NewIRCTCScraper(*outputDir)

	if err := scraper.Scrape(); err != nil {
		log.Fatalf("Failed to scrape IRCTC data: %v", err)
	}

	log.Printf("IRCTC data scraped successfully to %s", *outputDir)
	os.Exit(0)
}


