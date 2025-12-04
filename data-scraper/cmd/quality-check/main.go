package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type QualityReport struct {
	Dataset       string
	Files         map[string]FileReport
	Issues        []string
	Warnings      []string
	RecordCounts  map[string]int
}

type FileReport struct {
	FileName      string
	RecordCount   int
	ColumnCount   int
	Issues        []string
	Warnings      []string
	SampleRecords []map[string]string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gtfs_directory>")
		os.Exit(1)
	}

	gtfsDir := os.Args[1]
	report := checkGTFSQuality(gtfsDir)
	printReport(report)
}

func checkGTFSQuality(gtfsDir string) QualityReport {
	report := QualityReport{
		Dataset:      filepath.Base(gtfsDir),
		Files:        make(map[string]FileReport),
		Issues:       []string{},
		Warnings:     []string{},
		RecordCounts: make(map[string]int),
	}

	requiredFiles := []string{"agency.txt", "stops.txt", "routes.txt", "trips.txt", "stop_times.txt", "calendar.txt"}
	optionalFiles := []string{"shapes.txt", "fare_attributes.txt", "fare_rules.txt"}

	// Check required files
	for _, fileName := range requiredFiles {
		filePath := filepath.Join(gtfsDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			report.Issues = append(report.Issues, fmt.Sprintf("Missing required file: %s", fileName))
		} else {
			fileReport := checkFile(filePath, fileName)
			report.Files[fileName] = fileReport
			report.RecordCounts[fileName] = fileReport.RecordCount
		}
	}

	// Check optional files
	for _, fileName := range optionalFiles {
		filePath := filepath.Join(gtfsDir, fileName)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			fileReport := checkFile(filePath, fileName)
			report.Files[fileName] = fileReport
			report.RecordCounts[fileName] = fileReport.RecordCount
		}
	}

	// Cross-file validations
	validateRelationships(&report)

	return report
}

func checkFile(filePath, fileName string) FileReport {
	report := FileReport{
		FileName:      fileName,
		Issues:        []string{},
		Warnings:      []string{},
		SampleRecords: []map[string]string{},
	}

	file, err := os.Open(filePath)
	if err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("Cannot open file: %v", err))
		return report
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("Cannot read CSV: %v", err))
		return report
	}

	if len(records) < 2 {
		report.Issues = append(report.Issues, "File has no data records (only header)")
		return report
	}

	report.RecordCount = len(records) - 1 // Exclude header
	report.ColumnCount = len(records[0])
	headers := records[0]

	// Store sample records (first 3 data records)
	sampleCount := 3
	if report.RecordCount < sampleCount {
		sampleCount = report.RecordCount
	}
	for i := 1; i <= sampleCount && i < len(records); i++ {
		record := make(map[string]string)
		for j, header := range headers {
			if j < len(records[i]) {
				record[header] = records[i][j]
			}
		}
		report.SampleRecords = append(report.SampleRecords, record)
	}

	// File-specific validations
	switch fileName {
	case "agency.txt":
		validateAgency(records, &report)
	case "stops.txt":
		validateStops(records, &report)
	case "routes.txt":
		validateRoutes(records, &report)
	case "trips.txt":
		validateTrips(records, &report)
	case "stop_times.txt":
		validateStopTimes(records, &report)
	case "calendar.txt":
		validateCalendar(records, &report)
	case "shapes.txt":
		validateShapes(records, &report)
	}

	return report
}

func validateAgency(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Issues = append(report.Issues, "No agency records found")
		return
	}

	headers := records[0]
	agencyIDIdx := findIndex(headers, "agency_id")
	agencyNameIdx := findIndex(headers, "agency_name")
	agencyURLIdx := findIndex(headers, "agency_url")
	agencyTimezoneIdx := findIndex(headers, "agency_timezone")

	if agencyIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: agency_id")
	}
	if agencyNameIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: agency_name")
	}
	if agencyTimezoneIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: agency_timezone")
	}

	agencyIDs := make(map[string]bool)
	for i := 1; i < len(records); i++ {
		if agencyIDIdx >= 0 && agencyIDIdx < len(records[i]) {
			agencyID := strings.TrimSpace(records[i][agencyIDIdx])
			if agencyID == "" {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Empty agency_id", i+1))
			} else if agencyIDs[agencyID] {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Duplicate agency_id: %s", i+1, agencyID))
			} else {
				agencyIDs[agencyID] = true
			}
		}
		if agencyNameIdx >= 0 && agencyNameIdx < len(records[i]) {
			if strings.TrimSpace(records[i][agencyNameIdx]) == "" {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: Empty agency_name", i+1))
			}
		}
		if agencyURLIdx >= 0 && agencyURLIdx < len(records[i]) {
			url := strings.TrimSpace(records[i][agencyURLIdx])
			if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: Invalid URL format: %s", i+1, url))
			}
		}
	}
}

func validateStops(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Issues = append(report.Issues, "No stop records found")
		return
	}

	headers := records[0]
	stopIDIdx := findIndex(headers, "stop_id")
	stopLatIdx := findIndex(headers, "stop_lat")
	stopLonIdx := findIndex(headers, "stop_lon")
	stopNameIdx := findIndex(headers, "stop_name")

	if stopIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: stop_id")
	}
	if stopLatIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: stop_lat")
	}
	if stopLonIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: stop_lon")
	}

	stopIDs := make(map[string]bool)
	invalidCoords := 0
	for i := 1; i < len(records); i++ {
		if stopIDIdx >= 0 && stopIDIdx < len(records[i]) {
			stopID := strings.TrimSpace(records[i][stopIDIdx])
			if stopID == "" {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Empty stop_id", i+1))
			} else if stopIDs[stopID] {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Duplicate stop_id: %s", i+1, stopID))
			} else {
				stopIDs[stopID] = true
			}
		}

		if stopLatIdx >= 0 && stopLatIdx < len(records[i]) && stopLonIdx >= 0 && stopLonIdx < len(records[i]) {
			latStr := strings.TrimSpace(records[i][stopLatIdx])
			lonStr := strings.TrimSpace(records[i][stopLonIdx])
			if latStr != "" && lonStr != "" {
				lat, err1 := strconv.ParseFloat(latStr, 64)
				lon, err2 := strconv.ParseFloat(lonStr, 64)
				if err1 != nil || err2 != nil {
					invalidCoords++
				} else if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
					invalidCoords++
				} else if lat < 8 || lat > 37 || lon < 68 || lon > 97 {
					// Rough bounds for India - this is a warning, not an error
					report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: Coordinates outside India bounds: (%f, %f)", i+1, lat, lon))
				}
			}
		}

		if stopNameIdx >= 0 && stopNameIdx < len(records[i]) {
			if strings.TrimSpace(records[i][stopNameIdx]) == "" {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: Empty stop_name", i+1))
			}
		}
	}

	if invalidCoords > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("%d stops have invalid coordinates", invalidCoords))
	}
}

func validateRoutes(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Issues = append(report.Issues, "No route records found")
		return
	}

	headers := records[0]
	routeIDIdx := findIndex(headers, "route_id")
	routeTypeIdx := findIndex(headers, "route_type")

	if routeIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: route_id")
	}

	routeIDs := make(map[string]bool)
	for i := 1; i < len(records); i++ {
		if routeIDIdx >= 0 && routeIDIdx < len(records[i]) {
			routeID := strings.TrimSpace(records[i][routeIDIdx])
			if routeID == "" {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Empty route_id", i+1))
			} else if routeIDs[routeID] {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Duplicate route_id: %s", i+1, routeID))
			} else {
				routeIDs[routeID] = true
			}
		}

		if routeTypeIdx >= 0 && routeTypeIdx < len(records[i]) {
			routeTypeStr := strings.TrimSpace(records[i][routeTypeIdx])
			if routeTypeStr != "" {
				routeType, err := strconv.Atoi(routeTypeStr)
				if err != nil {
					report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: Invalid route_type: %s", i+1, routeTypeStr))
				} else if routeType < 0 || routeType > 7 {
					report.Warnings = append(report.Warnings, fmt.Sprintf("Row %d: route_type out of valid range (0-7): %d", i+1, routeType))
				}
			}
		}
	}
}

func validateTrips(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Issues = append(report.Issues, "No trip records found")
		return
	}

	headers := records[0]
	tripIDIdx := findIndex(headers, "trip_id")
	routeIDIdx := findIndex(headers, "route_id")
	serviceIDIdx := findIndex(headers, "service_id")

	if tripIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: trip_id")
	}
	if routeIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: route_id")
	}
	if serviceIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: service_id")
	}

	tripIDs := make(map[string]bool)
	for i := 1; i < len(records); i++ {
		if tripIDIdx >= 0 && tripIDIdx < len(records[i]) {
			tripID := strings.TrimSpace(records[i][tripIDIdx])
			if tripID == "" {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Empty trip_id", i+1))
			} else if tripIDs[tripID] {
				report.Issues = append(report.Issues, fmt.Sprintf("Row %d: Duplicate trip_id: %s", i+1, tripID))
			} else {
				tripIDs[tripID] = true
			}
		}
	}
}

func validateStopTimes(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Issues = append(report.Issues, "No stop_times records found")
		return
	}

	headers := records[0]
	tripIDIdx := findIndex(headers, "trip_id")
	stopIDIdx := findIndex(headers, "stop_id")
	arrivalTimeIdx := findIndex(headers, "arrival_time")
	departureTimeIdx := findIndex(headers, "departure_time")
	stopSequenceIdx := findIndex(headers, "stop_sequence")

	if tripIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: trip_id")
	}
	if stopIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: stop_id")
	}
	if arrivalTimeIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: arrival_time")
	}
	if departureTimeIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: departure_time")
	}
	if stopSequenceIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: stop_sequence")
	}

	invalidTimes := 0
	for i := 1; i < len(records) && i <= 1000; i++ { // Check first 1000 records
		if arrivalTimeIdx >= 0 && arrivalTimeIdx < len(records[i]) {
			timeStr := strings.TrimSpace(records[i][arrivalTimeIdx])
			if !isValidTime(timeStr) {
				invalidTimes++
			}
		}
		if departureTimeIdx >= 0 && departureTimeIdx < len(records[i]) {
			timeStr := strings.TrimSpace(records[i][departureTimeIdx])
			if !isValidTime(timeStr) {
				invalidTimes++
			}
		}
	}

	if invalidTimes > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Found %d invalid time formats (checked first 1000 records)", invalidTimes))
	}
}

func validateCalendar(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Warnings = append(report.Warnings, "No calendar records found (calendar_dates.txt might be used instead)")
		return
	}

	headers := records[0]
	serviceIDIdx := findIndex(headers, "service_id")
	mondayIdx := findIndex(headers, "monday")
	tuesdayIdx := findIndex(headers, "tuesday")

	if serviceIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: service_id")
	}
	if mondayIdx == -1 {
		report.Warnings = append(report.Warnings, "Missing monday column")
	}
	if tuesdayIdx == -1 {
		report.Warnings = append(report.Warnings, "Missing tuesday column")
	}
}

func validateShapes(records [][]string, report *FileReport) {
	if len(records) < 2 {
		report.Warnings = append(report.Warnings, "No shape records found")
		return
	}

	headers := records[0]
	shapeIDIdx := findIndex(headers, "shape_id")
	shapePtLatIdx := findIndex(headers, "shape_pt_lat")
	shapePtLonIdx := findIndex(headers, "shape_pt_lon")

	if shapeIDIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: shape_id")
	}
	if shapePtLatIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: shape_pt_lat")
	}
	if shapePtLonIdx == -1 {
		report.Issues = append(report.Issues, "Missing required column: shape_pt_lon")
	}
}

func validateRelationships(report *QualityReport) {
	// Check if route_ids in trips.txt exist in routes.txt
	if tripsReport, ok := report.Files["trips.txt"]; ok {
		if routesReport, ok := report.Files["routes.txt"]; ok {
			// This is a simplified check - in production, we'd parse and compare IDs
			if tripsReport.RecordCount > 0 && routesReport.RecordCount == 0 {
				report.Warnings = append(report.Warnings, "trips.txt has records but routes.txt is empty")
			}
		}
	}

	// Check if stop_ids in stop_times.txt exist in stops.txt
	if stopTimesReport, ok := report.Files["stop_times.txt"]; ok {
		if stopsReport, ok := report.Files["stops.txt"]; ok {
			if stopTimesReport.RecordCount > 0 && stopsReport.RecordCount == 0 {
				report.Warnings = append(report.Warnings, "stop_times.txt has records but stops.txt is empty")
			}
		}
	}
}

func isValidTime(timeStr string) bool {
	if timeStr == "" {
		return false
	}
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func findIndex(headers []string, column string) int {
	for i, h := range headers {
		if strings.ToLower(strings.TrimSpace(h)) == strings.ToLower(column) {
			return i
		}
	}
	return -1
}

func printReport(report QualityReport) {
	fmt.Println("=" + strings.Repeat("=", 78))
	fmt.Printf("GTFS Quality Check Report: %s\n", report.Dataset)
	fmt.Println("=" + strings.Repeat("=", 78))
	fmt.Println()

	// Summary
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("-", 80))
	for fileName, count := range report.RecordCounts {
		fmt.Printf("  %-20s: %10d records\n", fileName, count)
	}
	fmt.Println()

	// File Details
	fmt.Println("FILE DETAILS")
	fmt.Println(strings.Repeat("-", 80))
	for fileName, fileReport := range report.Files {
		fmt.Printf("\n%s (%d records, %d columns)\n", fileName, fileReport.RecordCount, fileReport.ColumnCount)
		if len(fileReport.Issues) > 0 {
			fmt.Println("  Issues:")
			for _, issue := range fileReport.Issues {
				fmt.Printf("    ❌ %s\n", issue)
			}
		}
		if len(fileReport.Warnings) > 0 {
			fmt.Println("  Warnings:")
			for _, warning := range fileReport.Warnings {
				fmt.Printf("    ⚠️  %s\n", warning)
			}
		}
		if len(fileReport.SampleRecords) > 0 {
			fmt.Println("  Sample Records:")
			for i, record := range fileReport.SampleRecords {
				if i >= 2 { // Show max 2 samples
					break
				}
				fmt.Printf("    Record %d:\n", i+1)
				for key, value := range record {
					if len(value) > 50 {
						value = value[:47] + "..."
					}
					fmt.Printf("      %s: %s\n", key, value)
				}
			}
		}
	}

	// Overall Issues
	if len(report.Issues) > 0 {
		fmt.Println("\nOVERALL ISSUES")
		fmt.Println(strings.Repeat("-", 80))
		for _, issue := range report.Issues {
			fmt.Printf("  ❌ %s\n", issue)
		}
	}

	// Overall Warnings
	if len(report.Warnings) > 0 {
		fmt.Println("\nOVERALL WARNINGS")
		fmt.Println(strings.Repeat("-", 80))
		for _, warning := range report.Warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
	}

	// Quality Score
	totalIssues := len(report.Issues)
	for _, fileReport := range report.Files {
		totalIssues += len(fileReport.Issues)
	}
	totalWarnings := len(report.Warnings)
	for _, fileReport := range report.Files {
		totalWarnings += len(fileReport.Warnings)
	}

	fmt.Println("\nQUALITY SCORE")
	fmt.Println(strings.Repeat("-", 80))
	if totalIssues == 0 && totalWarnings == 0 {
		fmt.Println("  ✅ EXCELLENT - No issues or warnings found")
	} else if totalIssues == 0 {
		fmt.Printf("  ✅ GOOD - No critical issues, %d warning(s)\n", totalWarnings)
	} else {
		fmt.Printf("  ⚠️  NEEDS ATTENTION - %d issue(s), %d warning(s)\n", totalIssues, totalWarnings)
	}
	fmt.Println()
}

