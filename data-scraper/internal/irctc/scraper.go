package irctc

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"data-scraper/internal/models"
)

type IRCTCScraper struct {
	outputDir string
	collector *colly.Collector
	client    *http.Client
}

func NewIRCTCScraper(outputDir string) *IRCTCScraper {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       3 * time.Second, // Be respectful
	})

	return &IRCTCScraper{
		outputDir: outputDir,
		collector: c,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Scrape scrapes IRCTC data and converts to GTFS
func (s *IRCTCScraper) Scrape() error {
	log.Println("Starting IRCTC scraper...")

	// Create output directory
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Scrape stations
	stations, err := s.scrapeStations()
	if err != nil {
		return fmt.Errorf("failed to scrape stations: %w", err)
	}

	// Scrape trains
	trains, err := s.scrapeTrains(stations)
	if err != nil {
		return fmt.Errorf("failed to scrape trains: %w", err)
	}

	// Convert to GTFS
	if err := s.convertToGTFS(stations, trains); err != nil {
		return fmt.Errorf("failed to convert to GTFS: %w", err)
	}

	log.Println("IRCTC scraping completed successfully")
	return nil
}

// scrapeStations scrapes railway stations
func (s *IRCTCScraper) scrapeStations() ([]models.IRCTCStation, error) {
	var stations []models.IRCTCStation

	// Try to scrape from IRCTC API or use hardcoded data
	// IRCTC doesn't have a public API, so we'll use hardcoded major stations
	log.Println("Using hardcoded IRCTC station data...")
	stations = s.getHardcodedStations()

	return stations, nil
}

// scrapeTrains scrapes train routes
func (s *IRCTCScraper) scrapeTrains(stations []models.IRCTCStation) ([]models.IRCTCTrain, error) {
	var trains []models.IRCTCTrain

	// Try to scrape train schedules
	// Since IRCTC requires authentication, we'll use hardcoded popular routes
	log.Println("Using hardcoded IRCTC train data...")
	trains = s.getHardcodedTrains(stations)

	return trains, nil
}

// convertToGTFS converts scraped data to GTFS format
func (s *IRCTCScraper) convertToGTFS(stations []models.IRCTCStation, trains []models.IRCTCTrain) error {
	// Write agency.txt
	if err := s.writeAgency(); err != nil {
		return err
	}

	// Write routes.txt
	if err := s.writeRoutes(trains); err != nil {
		return err
	}

	// Write stops.txt
	if err := s.writeStops(stations); err != nil {
		return err
	}

	// Write trips.txt and stop_times.txt
	if err := s.writeTripsAndStopTimes(trains); err != nil {
		return err
	}

	// Write calendar.txt
	if err := s.writeCalendar(); err != nil {
		return err
	}

	return nil
}

func (s *IRCTCScraper) writeAgency() error {
	file, err := os.Create(fmt.Sprintf("%s/agency.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"agency_id", "agency_name", "agency_url", "agency_timezone", "agency_lang"})

	writer.Write([]string{
		"IRCTC",
		"Indian Railway Catering and Tourism Corporation",
		"https://www.irctc.co.in",
		"Asia/Kolkata",
		"en",
	})

	return nil
}

func (s *IRCTCScraper) writeRoutes(trains []models.IRCTCTrain) error {
	file, err := os.Create(fmt.Sprintf("%s/routes.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"route_id", "agency_id", "route_short_name", "route_long_name", "route_type"})

	// Route type: 2 = Rail
	for _, train := range trains {
		writer.Write([]string{
			fmt.Sprintf("IRCTC_%s", train.TrainNumber),
			"IRCTC",
			train.TrainNumber,
			train.TrainName,
			"2", // Rail
		})
	}

	return nil
}

func (s *IRCTCScraper) writeStops(stations []models.IRCTCStation) error {
	file, err := os.Create(fmt.Sprintf("%s/stops.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"stop_id", "stop_code", "stop_name", "stop_lat", "stop_lon"})

	for _, station := range stations {
		writer.Write([]string{
			fmt.Sprintf("IRCTC_%s", station.StationCode),
			station.StationCode,
			station.StationName,
			fmt.Sprintf("%.6f", station.Latitude),
			fmt.Sprintf("%.6f", station.Longitude),
		})
	}

	return nil
}

func (s *IRCTCScraper) writeTripsAndStopTimes(trains []models.IRCTCTrain) error {
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

	tripsWriter.Write([]string{"route_id", "service_id", "trip_id", "trip_headsign"})
	stopTimesWriter.Write([]string{"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence"})

	// Create service IDs based on days
	serviceMap := map[string]string{
		"Daily":        "IRCTC_DAILY",
		"Monday":       "IRCTC_MON",
		"Tuesday":      "IRCTC_TUE",
		"Wednesday":    "IRCTC_WED",
		"Thursday":     "IRCTC_THU",
		"Friday":       "IRCTC_FRI",
		"Saturday":     "IRCTC_SAT",
		"Sunday":       "IRCTC_SUN",
	}

	tripCounter := 0

	for _, train := range trains {
		serviceID := "IRCTC_DAILY"
		if len(train.Days) > 0 {
			// Use first day as service ID (simplified)
			serviceID = serviceMap[train.Days[0].String()]
		}

		tripID := fmt.Sprintf("IRCTC_TRIP_%d", tripCounter)
		tripCounter++

		tripsWriter.Write([]string{
			fmt.Sprintf("IRCTC_%s", train.TrainNumber),
			serviceID,
			tripID,
			train.TrainName,
		})

		// Generate stop times (assume 30 minutes per station)
		hour := 6
		minute := 0
		for i, station := range train.Stations {
			arrivalTime := fmt.Sprintf("%02d:%02d:00", hour, minute)
			departureTime := fmt.Sprintf("%02d:%02d:00", hour, minute+2)

			stopTimesWriter.Write([]string{
				tripID,
				arrivalTime,
				departureTime,
				fmt.Sprintf("IRCTC_%s", station.StationCode),
				fmt.Sprintf("%d", i+1),
			})

			minute += 30
			if minute >= 60 {
				hour++
				minute -= 60
			}
		}
	}

	return nil
}

func (s *IRCTCScraper) writeCalendar() error {
	file, err := os.Create(fmt.Sprintf("%s/calendar.txt", s.outputDir))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"service_id", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday", "start_date", "end_date"})

	now := time.Now()
	startDate := now.Format("20060102")
	endDate := now.AddDate(1, 0, 0).Format("20060102")

	// Daily service
	writer.Write([]string{
		"IRCTC_DAILY",
		"1", "1", "1", "1", "1", "1", "1",
		startDate,
		endDate,
	})

	return nil
}

// Helper functions

func (s *IRCTCScraper) getHardcodedStations() []models.IRCTCStation {
	return []models.IRCTCStation{
		{StationCode: "SBC", StationName: "Bangalore City", Latitude: 12.9716, Longitude: 77.5946, Zone: "SWR"},
		{StationCode: "MAS", StationName: "Chennai Central", Latitude: 13.0827, Longitude: 80.2707, Zone: "SR"},
		{StationCode: "NDLS", StationName: "New Delhi", Latitude: 28.6448, Longitude: 77.2167, Zone: "NR"},
		{StationCode: "CSTM", StationName: "Mumbai CST", Latitude: 18.9400, Longitude: 72.8356, Zone: "CR"},
		{StationCode: "HWH", StationName: "Howrah", Latitude: 22.5958, Longitude: 88.3431, Zone: "ER"},
		{StationCode: "PNBE", StationName: "Patna Junction", Latitude: 25.6093, Longitude: 85.1235, Zone: "ECR"},
		{StationCode: "BCT", StationName: "Mumbai Central", Latitude: 18.9700, Longitude: 72.8197, Zone: "WR"},
		{StationCode: "ADI", StationName: "Ahmedabad", Latitude: 23.0225, Longitude: 72.5714, Zone: "WR"},
		{StationCode: "JP", StationName: "Jaipur", Latitude: 26.9221, Longitude: 75.7789, Zone: "NWR"},
		{StationCode: "HYB", StationName: "Hyderabad Deccan", Latitude: 17.3850, Longitude: 78.4867, Zone: "SCR"},
	}
}

func (s *IRCTCScraper) getHardcodedTrains(stations []models.IRCTCStation) []models.IRCTCTrain {
	stationMap := make(map[string]models.IRCTCStation)
	for _, s := range stations {
		stationMap[s.StationCode] = s
	}

	return []models.IRCTCTrain{
		{
			TrainNumber: "12627",
			TrainName:   "Bangalore Express",
			FromStation: "SBC",
			ToStation:   "MAS",
			Stations: []models.IRCTCStation{
				stationMap["SBC"],
				stationMap["MAS"],
			},
			Days: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday},
		},
		{
			TrainNumber: "12253",
			TrainName:   "Yesvantpur Duronto Express",
			FromStation: "YPR",
			ToStation:   "NDLS",
			Stations: []models.IRCTCStation{
				{StationCode: "YPR", StationName: "Yesvantpur", Latitude: 13.0225, Longitude: 77.5514, Zone: "SWR"},
				stationMap["NDLS"],
			},
			Days: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
		},
	}
}


