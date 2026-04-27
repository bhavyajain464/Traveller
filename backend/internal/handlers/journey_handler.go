package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type JourneyHandler struct {
	routePlanner services.JourneyPlanner
}

func NewJourneyHandler(routePlanner services.JourneyPlanner) *JourneyHandler {
	return &JourneyHandler{routePlanner: routePlanner}
}

func (h *JourneyHandler) PlanJourney(c *gin.Context) {
	var req models.JourneyRequest

	// Parse query parameters
	fromLatStr := c.Query("from_lat")
	fromLonStr := c.Query("from_lon")
	toLatStr := c.Query("to_lat")
	toLonStr := c.Query("to_lon")
	departureTimeStr := c.Query("departure_time")
	arrivalTimeStr := c.Query("arrival_time")
	dateStr := c.Query("date")

	if fromLatStr == "" || fromLonStr == "" || toLatStr == "" || toLonStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_lat, from_lon, to_lat, and to_lon are required"})
		return
	}

	fromLat, err := strconv.ParseFloat(fromLatStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_lat"})
		return
	}

	fromLon, err := strconv.ParseFloat(fromLonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_lon"})
		return
	}

	toLat, err := strconv.ParseFloat(toLatStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_lat"})
		return
	}

	toLon, err := strconv.ParseFloat(toLonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_lon"})
		return
	}

	req.FromLat = fromLat
	req.FromLon = fromLon
	req.ToLat = toLat
	req.ToLon = toLon

	// Parse date parameter (for day-specific service plans)
	var journeyDate time.Time
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		journeyDate = parsedDate
	} else {
		// Default to today if not specified
		journeyDate = time.Now()
	}
	req.Date = &journeyDate

	if departureTimeStr != "" {
		depTime, err := time.Parse(time.RFC3339, departureTimeStr)
		if err != nil {
			// Try parsing as HH:MM:SS
			depTime, err = time.Parse("15:04:05", departureTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_time format"})
				return
			}
			// Set to the specified date (or today if not specified)
			depTime = time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(),
				depTime.Hour(), depTime.Minute(), depTime.Second(), 0, journeyDate.Location())
		}
		req.DepartureTime = &depTime
	}

	if arrivalTimeStr != "" {
		arrTime, err := time.Parse(time.RFC3339, arrivalTimeStr)
		if err != nil {
			arrTime, err = time.Parse("15:04:05", arrivalTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid arrival_time format"})
				return
			}
			// Set to the specified date (or today if not specified)
			arrTime = time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(),
				arrTime.Hour(), arrTime.Minute(), arrTime.Second(), 0, journeyDate.Location())
		}
		req.ArrivalTime = &arrTime
	}

	options, err := h.routePlanner.PlanJourney(req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if len(options) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no journey options found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"options": options,
		"count":   len(options),
	})
}
