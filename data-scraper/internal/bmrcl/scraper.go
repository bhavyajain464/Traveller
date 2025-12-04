package bmrcl

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"data-scraper/internal/models"

	"github.com/gocolly/colly/v2"
)

type BMRCLScraper struct {
	outputDir    string
	collector    *colly.Collector
	googleAPIKey string
	httpClient   *http.Client
}

func NewBMRCLScraper(outputDir string) *BMRCLScraper {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	// Set rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       2 * time.Second,
	})

	// Set language preference to English
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		// Try to set cookie for English language if website uses it
		r.Headers.Set("Cookie", "lang=en; language=en")
	})

	// Get Google API key from environment variable
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		log.Println("Warning: GOOGLE_MAPS_API_KEY not set. Geocoding will be skipped.")
	}

	return &BMRCLScraper{
		outputDir:    outputDir,
		collector:    c,
		googleAPIKey: apiKey,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Scrape scrapes BMRCL data and converts to GTFS
func (s *BMRCLScraper) Scrape() error {
	log.Println("Starting BMRCL scraper...")

	// Create output directory
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Scrape stations
	stations, err := s.scrapeStations()
	if err != nil {
		return fmt.Errorf("failed to scrape stations: %w", err)
	}

	// Scrape routes
	routes, err := s.scrapeRoutes(stations)
	if err != nil {
		return fmt.Errorf("failed to scrape routes: %w", err)
	}

	// Convert to GTFS
	if err := s.convertToGTFS(stations, routes); err != nil {
		return fmt.Errorf("failed to convert to GTFS: %w", err)
	}

	log.Println("BMRCL scraping completed successfully")
	return nil
}

// scrapeStations scrapes metro stations from BMRCL metro-network page
func (s *BMRCLScraper) scrapeStations() ([]models.BMRCLStation, error) {
	var stations []models.BMRCLStation

	// Scrape from the metro-network page
	// Try English version first
	metroNetworkURLs := []string{
		"https://www.bmrc.co.in/en/metro-network/",
		"https://www.bmrc.co.in/metro-network/?lang=en",
		"https://www.bmrc.co.in/metro-network/",
	}

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	// Set language headers to request English content
	c.OnRequest(func(r *colly.Request) {
		log.Printf("Scraping from: %s\n", r.URL.String())
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		// Set cookie for English if website supports it
		r.Headers.Set("Cookie", "lang=en; language=en; locale=en")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error scraping: %v\n", err)
	})

	// Map to track stations by line
	stationsByLine := make(map[string][]models.BMRCLStation)

	// First, try to find station data in tables (most common structure)
	c.OnHTML("table", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, row *colly.HTMLElement) {
			// Look for station names in table cells
			row.ForEach("td", func(j int, cell *colly.HTMLElement) {
				stationName := strings.TrimSpace(cell.Text)
				stationName = cleanText(stationName)

				if isValidStationName(stationName) {
					lineName := s.detectLineFromContext(cell)

					// Geocode the station
					lat, lon, err := s.geocodeStation(stationName + ", Bangalore Metro, India")
					if err != nil {
						log.Printf("Warning: Could not geocode '%s': %v\n", stationName, err)
						return
					}

					// Check if already exists
					exists := false
					for _, existing := range stationsByLine[lineName] {
						if strings.EqualFold(existing.StationName, stationName) {
							exists = true
							break
						}
					}

					if !exists {
						stationCode := s.generateStationCode(stationName)
						station := models.BMRCLStation{
							StationID:   fmt.Sprintf("BMRCL_%s_%d", lineName, len(stationsByLine[lineName])+1),
							StationName: stationName,
							Line:        lineName,
							Latitude:    lat,
							Longitude:   lon,
							StationCode: stationCode,
							Order:       len(stationsByLine[lineName]) + 1,
						}

						stationsByLine[lineName] = append(stationsByLine[lineName], station)
						log.Printf("Found station: %s (%s Line) at %.6f, %.6f\n", stationName, lineName, lat, lon)
					}
				}
			})
		})
	})

	// Also try lists and other structures
	c.OnHTML("body", func(e *colly.HTMLElement) {
		// Look for station information in various HTML structures
		e.ForEach("ul.station-list li, ol.station-list li, .metro-station, [data-station], .station-item, .station-name", func(i int, h *colly.HTMLElement) {
			stationName := strings.TrimSpace(h.Text)
			stationName = strings.ReplaceAll(stationName, "\n", " ")
			stationName = strings.ReplaceAll(stationName, "\t", " ")
			stationName = strings.TrimSpace(stationName)

			// Skip if empty or too short
			if stationName == "" || len(stationName) < 3 {
				return
			}

			// Skip if too long (likely not a station name)
			if len(stationName) > 50 {
				return
			}

			// Skip navigation/menu items (common patterns)
			skipPatterns := []string{
				"Home", "About", "Contact", "Login", "Register", "Search", "Menu", "Close", "Copyright",
				"Cookie", "Privacy", "Terms", "Policy", "Help", "FAQ", "Tender", "News", "Gallery",
				"ಮುಖಪುಟ", "ಸಂಪರ್ಕಿಸಿ", "ಹಕ್ಕು", "ನೀತಿ", "ಸಹಾಯ", // Kannada skip words
			}

			stationLower := strings.ToLower(stationName)
			for _, pattern := range skipPatterns {
				if strings.Contains(stationLower, strings.ToLower(pattern)) {
					return
				}
			}

			// Skip if contains numbers that look like dates, IDs, or measurements
			if strings.Contains(stationName, "202") || // Years
				strings.Contains(stationName, "km") ||
				strings.Contains(stationName, "km/h") ||
				strings.Contains(stationName, "Volt") ||
				strings.Contains(stationName, "ವೋಲ್ಟ್") {
				return
			}

			// Station names typically don't contain these patterns
			if strings.Contains(stationName, "://") || // URLs
				strings.Contains(stationName, "@") || // Email
				strings.Contains(stationName, "www.") {
				return
			}

			// Try to determine which line this station belongs to
			lineName := s.detectLineFromContext(h)

			// Try to extract coordinates from data attributes
			latStr := h.Attr("data-lat")
			lonStr := h.Attr("data-lon")
			if latStr == "" {
				latStr = h.Attr("data-latitude")
			}
			if lonStr == "" {
				lonStr = h.Attr("data-longitude")
			}

			lat, _ := strconv.ParseFloat(latStr, 64)
			lon, _ := strconv.ParseFloat(lonStr, 64)

			// If no coordinates found, geocode using Google Maps API
			if lat == 0 || lon == 0 {
				geocodedLat, geocodedLon, err := s.geocodeStation(stationName + ", Bangalore, India")
				if err != nil {
					log.Printf("Warning: Could not geocode station %s: %v\n", stationName, err)
					return
				}
				lat = geocodedLat
				lon = geocodedLon
			}

			// Check if station already exists
			exists := false
			for _, existing := range stationsByLine[lineName] {
				if strings.EqualFold(existing.StationName, stationName) {
					exists = true
					break
				}
			}

			if !exists && lat != 0 && lon != 0 {
				stationCode := s.generateStationCode(stationName)
				station := models.BMRCLStation{
					StationID:   fmt.Sprintf("BMRCL_%s_%d", lineName, len(stationsByLine[lineName])+1),
					StationName: stationName,
					Line:        lineName,
					Latitude:    lat,
					Longitude:   lon,
					StationCode: stationCode,
					Order:       len(stationsByLine[lineName]) + 1,
				}

				stationsByLine[lineName] = append(stationsByLine[lineName], station)
				log.Printf("Found station: %s (%s Line) at %.6f, %.6f\n", stationName, lineName, lat, lon)
			}
		})

		// Method 2: Look for JavaScript data or JSON embedded in page
		e.ForEach("script", func(i int, h *colly.HTMLElement) {
			scriptContent := h.Text
			// Try to extract station data from JavaScript variables or JSON
			if strings.Contains(scriptContent, "station") || strings.Contains(scriptContent, "metro") {
				// Look for JSON data structures
				if strings.Contains(scriptContent, "[{") || strings.Contains(scriptContent, "stations") {
					// Try to parse JSON from script
					// This would need more sophisticated parsing based on actual page structure
				}
			}
		})
	})

	// Visit the metro-network page (try multiple URLs)
	var lastErr error
	for _, url := range metroNetworkURLs {
		if err := c.Visit(url); err != nil {
			lastErr = err
			log.Printf("Failed to visit %s: %v, trying next URL...\n", url, err)
			continue
		}
		// If we got stations, break
		if len(stations) > 0 {
			break
		}
	}

	if len(stations) == 0 && lastErr != nil {
		return nil, fmt.Errorf("failed to visit metro-network page: %w", lastErr)
	}

	// Combine all stations
	for _, lineStations := range stationsByLine {
		stations = append(stations, lineStations...)
	}

	if len(stations) == 0 {
		return nil, fmt.Errorf("failed to scrape any stations. Website structure may have changed or page is not accessible")
	}

	log.Printf("Total stations scraped: %d\n", len(stations))
	return stations, nil
}

// detectLineFromContext tries to determine which metro line a station belongs to
func (s *BMRCLScraper) detectLineFromContext(e *colly.HTMLElement) string {
	// Check parent elements for line indicators
	parentHTML := e.DOM.Parent().Text()
	parentClass := e.DOM.Parent().AttrOr("class", "")
	parentID := e.DOM.Parent().AttrOr("id", "")

	// Also check the element itself
	elementClass := e.Attr("class")
	elementID := e.Attr("id")

	text := strings.ToLower(parentHTML + " " + parentClass + " " + parentID + " " + elementClass + " " + elementID)

	if strings.Contains(text, "green") || strings.Contains(text, "ಹಸಿರು") {
		return "Green"
	}
	if strings.Contains(text, "purple") || strings.Contains(text, "ನೇರಳೆ") {
		return "Purple"
	}
	if strings.Contains(text, "blue") || strings.Contains(text, "ನೀಲಿ") {
		return "Blue"
	}

	// Default to Green if cannot determine
	return "Green"
}

// detectLineFromTableContext tries to determine line from table context
func (s *BMRCLScraper) detectLineFromTableContext(table, row, cell *colly.HTMLElement) string {
	// Check table caption, headers, or nearby elements
	tableHTML := table.DOM.Text()
	rowHTML := row.DOM.Text()

	text := strings.ToLower(tableHTML + " " + rowHTML)

	if strings.Contains(text, "green") || strings.Contains(text, "ಹಸಿರು") {
		return "Green"
	}
	if strings.Contains(text, "purple") || strings.Contains(text, "ನೇರಳೆ") {
		return "Purple"
	}
	if strings.Contains(text, "blue") || strings.Contains(text, "ನೀಲಿ") {
		return "Blue"
	}

	// Check if table is in a section with line name
	tableParent := table.DOM.Parent()
	if tableParent != nil {
		parentText := strings.ToLower(tableParent.Text())
		if strings.Contains(parentText, "green") || strings.Contains(parentText, "ಹಸಿರು") {
			return "Green"
		}
		if strings.Contains(parentText, "purple") || strings.Contains(parentText, "ನೇರಳೆ") {
			return "Purple"
		}
		if strings.Contains(parentText, "blue") || strings.Contains(parentText, "ನೀಲಿ") {
			return "Blue"
		}
	}

	// Default to Green if cannot determine
	return "Green"
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// geocodeStation uses Google Maps Geocoding API to get coordinates
func (s *BMRCLScraper) geocodeStation(address string) (float64, float64, error) {
	if s.googleAPIKey == "" {
		return 0, 0, fmt.Errorf("Google Maps API key not configured")
	}

	// Build geocoding API URL
	apiURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Add("address", address)
	params.Add("key", s.googleAPIKey)

	reqURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	resp, err := s.httpClient.Get(reqURL)
	if err != nil {
		return 0, 0, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("geocoding API returned status %d", resp.StatusCode)
	}

	var result struct {
		Status  string `json:"status"`
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if result.Status != "OK" || len(result.Results) == 0 {
		return 0, 0, fmt.Errorf("geocoding failed: %s", result.Status)
	}

	location := result.Results[0].Geometry.Location
	log.Printf("Geocoded '%s' to %.6f, %.6f\n", address, location.Lat, location.Lng)

	// Rate limiting - be respectful to Google API
	time.Sleep(100 * time.Millisecond)

	return location.Lat, location.Lng, nil
}

// cleanText removes extra whitespace and cleans up text
func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	// Remove multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return strings.TrimSpace(text)
}

// isValidStationName checks if a string looks like a valid station name
func isValidStationName(name string) bool {
	// Skip if empty or too short
	if name == "" || len(name) < 3 {
		return false
	}

	// Skip if too long (likely not a station name)
	if len(name) > 50 {
		return false
	}

	// Skip navigation/menu items (both English and Kannada)
	skipPatterns := []string{
		// English
		"home", "about", "contact", "login", "register", "search", "menu", "close", "copyright",
		"cookie", "privacy", "terms", "policy", "help", "faq", "tender", "news", "gallery",
		"station", "metro", "line", "route", "map", "network", "project", "phase",
		// Kannada common words
		"ಮುಖಪುಟ", "ಸಂಪರ್ಕಿಸಿ", "ಹಕ್ಕು", "ನೀತಿ", "ಸಹಾಯ", "ವಾರ್ತೆ", "ಗ್ಯಾಲರಿ",
		"ಮೆಟ್ರೋ", "ನಿಲ್ದಾಣ", "ಮಾರ್ಗ", "ನಕ್ಷೆ", "ಯೋಜನೆ",
		// Technical terms
		"km", "km/h", "volt", "ವೋಲ್ಟ್", "ಕಿ.ಮೀ", "202", "://", "@", "www.",
		"gauge", "traction", "signal", "train", "car", "set",
	}

	nameLower := strings.ToLower(name)
	for _, pattern := range skipPatterns {
		if strings.Contains(nameLower, pattern) {
			return false
		}
	}

	// Station names typically don't start with numbers (unless it's a route number)
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' && !strings.Contains(name, " ") {
		return false
	}

	// Check if it looks like a station name (contains common station name patterns)
	// Station names often end with words like "Road", "Station", "Circle", "Nagar", etc.
	stationSuffixes := []string{
		"road", "station", "circle", "nagar", "layout", "colony", "park", "market",
		"cross", "junction", "terminal", "bus stand", "railway station",
		// Kannada equivalents
		"ರಸ್ತೆ", "ನಿಲ್ದಾಣ", "ವೃತ್ತ", "ನಗರ", "ಪಾರ್ಕ್", "ಮಾರುಕಟ್ಟೆ", "ಕ್ರಾಸ್",
	}

	for _, suffix := range stationSuffixes {
		if strings.HasSuffix(nameLower, suffix) || strings.Contains(nameLower, " "+suffix) {
			return true // Has station suffix, likely valid
		}
	}

	// If it doesn't have a station suffix, it might still be valid if it's a known station name
	// For now, accept it if it passes other filters
	return true
}

// containsKannada checks if a string contains Kannada characters
func containsKannada(text string) bool {
	for _, r := range text {
		// Kannada Unicode range: U+0C80 to U+0CFF
		if r >= 0x0C80 && r <= 0x0CFF {
			return true
		}
	}
	return false
}

// scrapeRoutes scrapes metro routes
func (s *BMRCLScraper) scrapeRoutes(stations []models.BMRCLStation) ([]models.BMRCLRoute, error) {
	var routes []models.BMRCLRoute

	// Group stations by line
	stationsByLine := make(map[string][]models.BMRCLStation)
	for _, station := range stations {
		stationsByLine[station.Line] = append(stationsByLine[station.Line], station)
	}

	// Create routes for each line (bidirectional)
	for lineName, lineStations := range stationsByLine {
		if len(lineStations) < 2 {
			continue
		}

		// Sort stations by order
		for i := 0; i < len(lineStations)-1; i++ {
			for j := i + 1; j < len(lineStations); j++ {
				if lineStations[i].Order > lineStations[j].Order {
					lineStations[i], lineStations[j] = lineStations[j], lineStations[i]
				}
			}
		}

		// Forward route
		routeID := fmt.Sprintf("BMRCL_%s_UP", lineName)
		routes = append(routes, models.BMRCLRoute{
			RouteID:     routeID,
			RouteName:   fmt.Sprintf("%s Line (Up)", lineName),
			FromStation: lineStations[0].StationID,
			ToStation:   lineStations[len(lineStations)-1].StationID,
			Line:        lineName,
			Stations:    lineStations,
		})

		// Reverse route
		reverseStations := make([]models.BMRCLStation, len(lineStations))
		for i := range lineStations {
			reverseStations[i] = lineStations[len(lineStations)-1-i]
		}

		routeID = fmt.Sprintf("BMRCL_%s_DOWN", lineName)
		routes = append(routes, models.BMRCLRoute{
			RouteID:     routeID,
			RouteName:   fmt.Sprintf("%s Line (Down)", lineName),
			FromStation: reverseStations[0].StationID,
			ToStation:   reverseStations[len(reverseStations)-1].StationID,
			Line:        lineName,
			Stations:    reverseStations,
		})
	}

	return routes, nil
}

// convertToGTFS converts scraped data to GTFS format
func (s *BMRCLScraper) convertToGTFS(stations []models.BMRCLStation, routes []models.BMRCLRoute) error {
	// Write agency.txt
	if err := s.writeAgency(); err != nil {
		return err
	}

	// Write routes.txt
	if err := s.writeRoutes(routes); err != nil {
		return err
	}

	// Write stops.txt
	if err := s.writeStops(stations); err != nil {
		return err
	}

	// Write trips.txt and stop_times.txt
	if err := s.writeTripsAndStopTimes(routes); err != nil {
		return err
	}

	// Write calendar.txt
	if err := s.writeCalendar(); err != nil {
		return err
	}

	return nil
}

func (s *BMRCLScraper) writeAgency() error {
	file, err := os.Create(fmt.Sprintf("%s/agency.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"agency_id", "agency_name", "agency_url", "agency_timezone", "agency_lang"})

	// Write BMRCL agency
	writer.Write([]string{
		"BMRCL",
		"Bangalore Metro Rail Corporation Limited",
		"https://www.bmrc.co.in",
		"Asia/Kolkata",
		"en",
	})

	return nil
}

func (s *BMRCLScraper) writeRoutes(routes []models.BMRCLRoute) error {
	file, err := os.Create(fmt.Sprintf("%s/routes.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"route_id", "agency_id", "route_short_name", "route_long_name", "route_type", "route_color"})

	// Route type: 1 = Subway/Metro
	routeColors := map[string]string{
		"Green":  "008000",
		"Purple": "800080",
		"Blue":   "0000FF",
	}

	for _, route := range routes {
		color := routeColors[route.Line]
		if color == "" {
			color = "000000"
		}

		writer.Write([]string{
			route.RouteID,
			"BMRCL",
			route.Line,
			route.RouteName,
			"1", // Subway
			color,
		})
	}

	return nil
}

func (s *BMRCLScraper) writeStops(stations []models.BMRCLStation) error {
	file, err := os.Create(fmt.Sprintf("%s/stops.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"stop_id", "stop_code", "stop_name", "stop_lat", "stop_lon", "location_type"})

	for _, station := range stations {
		writer.Write([]string{
			station.StationID,
			station.StationCode,
			station.StationName,
			fmt.Sprintf("%.6f", station.Latitude),
			fmt.Sprintf("%.6f", station.Longitude),
			"0", // Stop
		})
	}

	return nil
}

func (s *BMRCLScraper) writeTripsAndStopTimes(routes []models.BMRCLRoute) error {
	tripsFile, err := os.Create(fmt.Sprintf("%s/trips.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer tripsFile.Close()

	stopTimesFile, err := os.Create(fmt.Sprintf("%s/stop_times.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer stopTimesFile.Close()

	tripsWriter := csv.NewWriter(tripsFile)
	defer tripsWriter.Flush()

	stopTimesWriter := csv.NewWriter(stopTimesFile)
	defer stopTimesWriter.Flush()

	// Write headers
	tripsWriter.Write([]string{"route_id", "service_id", "trip_id", "trip_headsign"})
	stopTimesWriter.Write([]string{"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence"})

	serviceID := "BMRCL_WEEKDAY"
	tripCounter := 0

	// Generate trips for each route (every 5 minutes during peak hours)
	for _, route := range routes {
		// Generate trips from 6 AM to 11 PM
		for hour := 6; hour < 23; hour++ {
			for minute := 0; minute < 60; minute += 5 {
				tripID := fmt.Sprintf("BMRCL_TRIP_%d", tripCounter)
				tripCounter++

				tripsWriter.Write([]string{
					route.RouteID,
					serviceID,
					tripID,
					route.RouteName,
				})

				// Generate stop times (assume 2 minutes per station)
				for i, station := range route.Stations {
					arrivalTime := fmt.Sprintf("%02d:%02d:00", hour, minute+i*2)
					departureTime := fmt.Sprintf("%02d:%02d:00", hour, minute+i*2+1)

					stopTimesWriter.Write([]string{
						tripID,
						arrivalTime,
						departureTime,
						station.StationID,
						fmt.Sprintf("%d", i+1),
					})
				}
			}
		}
	}

	return nil
}

func (s *BMRCLScraper) writeCalendar() error {
	file, err := os.Create(fmt.Sprintf("%s/calendar.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"service_id", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday", "start_date", "end_date"})

	// Weekday service (Monday-Friday)
	now := time.Now()
	startDate := now.Format("20060102")
	endDate := now.AddDate(1, 0, 0).Format("20060102")

	writer.Write([]string{
		"BMRCL_WEEKDAY",
		"1", "1", "1", "1", "1", "0", "0", // Mon-Fri
		startDate,
		endDate,
	})

	// Weekend service (Saturday-Sunday)
	writer.Write([]string{
		"BMRCL_WEEKEND",
		"0", "0", "0", "0", "0", "1", "1", // Sat-Sun
		startDate,
		endDate,
	})

	return nil
}

// Helper functions

func (s *BMRCLScraper) generateStationCode(stationName string) string {
	// Generate station code from name (e.g., "MG Road" -> "MGRD")
	parts := strings.Fields(strings.ToUpper(stationName))
	code := ""
	for _, part := range parts {
		if len(part) > 0 {
			code += string(part[0])
		}
	}
	if len(code) > 4 {
		code = code[:4]
	}
	return code
}
