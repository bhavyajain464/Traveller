package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type cleaner struct {
	inputDir  string
	outputDir string
	prefix    string

	routes   map[string]struct{}
	stops    map[string]struct{}
	services map[string]struct{}
	trips    map[string]struct{}
	fares    map[string]struct{}
	shapes   map[string]struct{}

	counts   map[string]int
	warnings []string
}

func main() {
	inputDir := flag.String("in", "", "Input GTFS directory")
	outputDir := flag.String("out", "", "Output cleaned GTFS directory")
	prefix := flag.String("prefix", "", "Feed-local ID prefix, e.g. bus or metro")
	flag.Parse()

	if *inputDir == "" || *outputDir == "" || *prefix == "" {
		log.Fatal("usage: go run cmd/clean-gtfs/main.go -in <gtfs-dir> -out <clean-dir> -prefix <feed-prefix>")
	}

	c := &cleaner{
		inputDir:  *inputDir,
		outputDir: *outputDir,
		prefix:    strings.TrimSuffix(*prefix, ":"),
		routes:    make(map[string]struct{}),
		stops:     make(map[string]struct{}),
		services:  make(map[string]struct{}),
		trips:     make(map[string]struct{}),
		fares:     make(map[string]struct{}),
		shapes:    make(map[string]struct{}),
		counts:    make(map[string]int),
	}

	if err := os.RemoveAll(c.outputDir); err != nil {
		log.Fatalf("failed to reset output directory: %v", err)
	}
	if err := os.MkdirAll(c.outputDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	steps := []string{
		"agency.txt",
		"routes.txt",
		"stops.txt",
		"calendar.txt",
		"trips.txt",
		"stop_times.txt",
		"shapes.txt",
		"fare_attributes.txt",
		"fare_rules.txt",
	}

	for _, fileName := range steps {
		if err := c.cleanFile(fileName); err != nil {
			log.Fatalf("failed cleaning %s: %v", fileName, err)
		}
	}

	log.Printf("Cleaned GTFS feed %q into %s", c.prefix, c.outputDir)
	for _, fileName := range steps {
		if count, ok := c.counts[fileName]; ok {
			log.Printf("  %s: %d rows", fileName, count)
		}
	}
	if len(c.warnings) > 0 {
		log.Printf("Warnings: %d", len(c.warnings))
		for i, warning := range c.warnings {
			if i >= 20 {
				log.Printf("  ... %d more warnings", len(c.warnings)-i)
				break
			}
			log.Printf("  - %s", warning)
		}
	}
}

func (c *cleaner) cleanFile(fileName string) error {
	inputPath := filepath.Join(c.inputDir, fileName)
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return nil
	}

	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer input.Close()

	outputPath := filepath.Join(c.outputDir, fileName)
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	writer := csv.NewWriter(output)
	defer writer.Flush()

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read headers: %w", err)
	}
	headers = trimRecord(headers)
	headerIndex := make(map[string]int, len(headers))
	for i, header := range headers {
		headerIndex[header] = i
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		record = normalizeRecord(record, len(headers))

		c.cleanRecord(fileName, headerIndex, record, line)

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write line %d: %w", line, err)
		}
		c.counts[fileName]++
	}

	return writer.Error()
}

func (c *cleaner) cleanRecord(fileName string, headerIndex map[string]int, record []string, line int) {
	switch fileName {
	case "routes.txt":
		c.namespace(headerIndex, record, "route_id")
		c.add(c.routes, value(headerIndex, record, "route_id"))
	case "stops.txt":
		c.namespace(headerIndex, record, "stop_id")
		c.namespace(headerIndex, record, "parent_station")
		c.namespace(headerIndex, record, "zone_id")
		c.add(c.stops, value(headerIndex, record, "stop_id"))
		c.validateLatLon(fileName, headerIndex, record, line, "stop_lat", "stop_lon")
	case "calendar.txt":
		c.namespace(headerIndex, record, "service_id")
		c.add(c.services, value(headerIndex, record, "service_id"))
	case "trips.txt":
		c.namespace(headerIndex, record, "route_id")
		c.namespace(headerIndex, record, "service_id")
		c.namespace(headerIndex, record, "trip_id")
		c.namespace(headerIndex, record, "shape_id")
		routeID := value(headerIndex, record, "route_id")
		serviceID := value(headerIndex, record, "service_id")
		tripID := value(headerIndex, record, "trip_id")
		c.warnMissing(c.routes, routeID, fileName, line, "route_id")
		c.warnMissing(c.services, serviceID, fileName, line, "service_id")
		c.add(c.trips, tripID)
	case "stop_times.txt":
		c.namespace(headerIndex, record, "trip_id")
		c.namespace(headerIndex, record, "stop_id")
		c.warnMissing(c.trips, value(headerIndex, record, "trip_id"), fileName, line, "trip_id")
		c.warnMissing(c.stops, value(headerIndex, record, "stop_id"), fileName, line, "stop_id")
	case "shapes.txt":
		c.namespace(headerIndex, record, "shape_id")
		c.add(c.shapes, value(headerIndex, record, "shape_id"))
		c.validateLatLon(fileName, headerIndex, record, line, "shape_pt_lat", "shape_pt_lon")
	case "fare_attributes.txt":
		c.namespace(headerIndex, record, "fare_id")
		c.add(c.fares, value(headerIndex, record, "fare_id"))
	case "fare_rules.txt":
		c.namespace(headerIndex, record, "fare_id")
		c.namespace(headerIndex, record, "route_id")
		c.namespace(headerIndex, record, "origin_id")
		c.namespace(headerIndex, record, "destination_id")
		c.namespace(headerIndex, record, "contains_id")
		c.warnMissing(c.fares, value(headerIndex, record, "fare_id"), fileName, line, "fare_id")
		routeID := value(headerIndex, record, "route_id")
		if routeID != "" {
			c.warnMissing(c.routes, routeID, fileName, line, "route_id")
		}
	}
}

func (c *cleaner) namespace(headerIndex map[string]int, record []string, field string) {
	idx, ok := headerIndex[field]
	if !ok || idx >= len(record) || record[idx] == "" {
		return
	}
	prefix := c.prefix + ":"
	if strings.HasPrefix(record[idx], prefix) {
		return
	}
	record[idx] = prefix + record[idx]
}

func (c *cleaner) add(set map[string]struct{}, item string) {
	if item == "" {
		return
	}
	set[item] = struct{}{}
}

func (c *cleaner) warnMissing(set map[string]struct{}, item, fileName string, line int, field string) {
	if item == "" {
		return
	}
	if _, ok := set[item]; !ok {
		c.warnings = append(c.warnings, fmt.Sprintf("%s:%d references missing %s=%s", fileName, line, field, item))
	}
}

func (c *cleaner) validateLatLon(fileName string, headerIndex map[string]int, record []string, line int, latField, lonField string) {
	lat := value(headerIndex, record, latField)
	lon := value(headerIndex, record, lonField)
	if lat == "" || lon == "" {
		c.warnings = append(c.warnings, fmt.Sprintf("%s:%d missing coordinates", fileName, line))
	}
}

func value(headerIndex map[string]int, record []string, field string) string {
	idx, ok := headerIndex[field]
	if !ok || idx >= len(record) {
		return ""
	}
	return record[idx]
}

func trimRecord(record []string) []string {
	out := make([]string, len(record))
	for i, value := range record {
		out[i] = strings.TrimSpace(value)
	}
	return out
}

func normalizeRecord(record []string, width int) []string {
	out := trimRecord(record)
	if len(out) >= width {
		return out
	}
	for len(out) < width {
		out = append(out, "")
	}
	return out
}
