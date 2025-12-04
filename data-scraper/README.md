# Traveller - Data Scraper

This package scrapes transit data from BMRCL (Bangalore Metro) and IRCTC (Indian Railways) and converts it to GTFS format for use with the Traveller backend service.

## Overview

The scraper collects:
- **BMRCL**: Metro routes, stations, schedules, fares
- **IRCTC**: Train routes, stations, schedules, fares

And converts to GTFS format compatible with the backend GTFS loader.

## Prerequisites

### Google Maps API Key (Required for BMRCL scraper)

The BMRCL scraper uses Google Maps Geocoding API to get station coordinates. You need to:

1. Get a Google Maps API key from [Google Cloud Console](https://console.cloud.google.com/)
2. Enable the "Geocoding API" for your project
3. Set it as an environment variable:

```bash
export GOOGLE_MAPS_API_KEY="your-api-key-here"
```

Or create a `.env` file:
```
GOOGLE_MAPS_API_KEY=your-api-key-here
```

## Installation

```bash
cd data-scraper
go mod download
```

## Usage

### BMRCL Scraper

```bash
# Set Google Maps API key
export GOOGLE_MAPS_API_KEY="your-api-key"

# Run scraper
go run cmd/bmrcl-scraper/main.go -output output/bmrcl
```

Scrapes:
- Metro lines (Green Line, Purple Line, etc.)
- Stations with coordinates (via Google Maps Geocoding)
- Routes and schedules
- Fare information

Output: `output/bmrcl/` directory with GTFS files

### IRCTC Scraper

```bash
go run cmd/irctc-scraper/main.go -output output/irctc
```

Scrapes:
- Train routes
- Railway stations
- Train schedules
- Fare information

Output: `output/irctc/` directory with GTFS files

### GTFS Converter

```bash
go run cmd/gtfs-converter/main.go --source output/bmrcl --target output/gtfs
```

Converts scraped data to standard GTFS format.

## Configuration

Create `.env` file (optional):

```env
# Google Maps API (Required for BMRCL)
GOOGLE_MAPS_API_KEY=your-api-key-here

# Output
OUTPUT_DIR=output
```

## Data Sources

### BMRCL
- **Source**: https://www.bmrc.co.in/metro-network/
- **Geocoding**: Google Maps Geocoding API
- Station information
- Route maps
- Fare charts
- Timetables

### IRCTC
- Official website: https://www.irctc.co.in
- Train schedules
- Station codes
- Route information
- Fare information

## Output Format

All scrapers output GTFS-compatible CSV files:
- `agency.txt`
- `routes.txt`
- `stops.txt`
- `trips.txt`
- `stop_times.txt`
- `calendar.txt`
- `fares.txt` (custom extension)

## Notes

- **No Hardcoded Data**: All data is scraped from live websites
- **Google Maps Integration**: Station coordinates are geocoded using Google Maps API
- Scrapers respect rate limits and robots.txt
- Data is cached to avoid repeated requests
- GTFS validation is performed before output
- Supports incremental updates

## Troubleshooting

### BMRCL Scraper Issues

1. **No stations found**: 
   - Check if website structure has changed
   - Verify network connectivity
   - Check if Google Maps API key is set correctly

2. **Geocoding failures**:
   - Verify Google Maps API key is valid
   - Check API quota/limits
   - Ensure Geocoding API is enabled

3. **Rate limiting**:
   - Google Maps API has rate limits
   - Scraper includes delays between requests
   - Consider upgrading API quota if needed

## Integration with Backend

After scraping, load data into backend:

```bash
# Load BMRCL data
cd backend
go run cmd/loader/main.go -data ../data-scraper/output/bmrcl

# Load IRCTC data
go run cmd/loader/main.go -data ../data-scraper/output/irctc

# Or aggregate multiple feeds
go run cmd/loader/main.go -data ../data-scraper/output/bmrcl,../data-scraper/output/irctc
```

## License

MIT
