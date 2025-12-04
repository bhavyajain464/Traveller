package main

import (
	"flag"
	"log"
	"os"

	"data-scraper/internal/bmrcl"
)

func main() {
	outputDir := flag.String("output", "output/bmrcl", "Output directory for GTFS files")
	flag.Parse()

	scraper := bmrcl.NewBMRCLScraper(*outputDir)

	if err := scraper.Scrape(); err != nil {
		log.Fatalf("Failed to scrape BMRCL data: %v", err)
	}

	log.Printf("BMRCL data scraped successfully to %s", *outputDir)
	os.Exit(0)
}


