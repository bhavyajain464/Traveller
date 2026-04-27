package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type V3Handler struct {
	journeyService *services.V3JourneyService
}

func NewV3Handler(journeyService *services.V3JourneyService) *V3Handler {
	return &V3Handler{journeyService: journeyService}
}

func (h *V3Handler) Locations(c *gin.Context) {
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		query = strings.TrimSpace(c.Query("q"))
	}
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	limit := 8
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	response, err := h.journeyService.SearchLocations(query, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *V3Handler) Journey(c *gin.Context) {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required"})
		return
	}

	mode := strings.TrimSpace(c.DefaultQuery("mode", "departure"))
	rawTime := strings.TrimSpace(c.Query("time"))
	if rawTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "time is required"})
		return
	}

	journeyTime, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		if fallback, parseErr := time.Parse("2006-01-02T15:04", rawTime); parseErr == nil {
			journeyTime = fallback
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "time must be ISO 8601"})
			return
		}
	}

	results := 5
	if rawResults := c.Query("results"); rawResults != "" {
		if parsed, parseErr := strconv.Atoi(rawResults); parseErr == nil && parsed > 0 && parsed <= 20 {
			results = parsed
		}
	}

	transportations := c.QueryArray("transportations[]")
	if len(transportations) == 0 {
		transportations = c.QueryArray("transportations")
	}

	response, err := h.journeyService.PlanJourney(services.V3JourneyQuery{
		From:            from,
		To:              to,
		Time:            journeyTime,
		Mode:            mode,
		Results:         results,
		Transportations: transportations,
	})
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "no journey options found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *V3Handler) Stationboard(c *gin.Context) {
	stationID := strings.TrimSpace(c.Query("station"))
	if stationID == "" {
		stationID = strings.TrimSpace(c.Query("id"))
	}
	if stationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "station is required"})
		return
	}

	limit := 10
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	boardTime := time.Now()
	if rawTime := strings.TrimSpace(c.Query("time")); rawTime != "" {
		parsed, err := time.Parse(time.RFC3339, rawTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "time must be ISO 8601"})
			return
		}
		boardTime = parsed
	}

	response, err := h.journeyService.Stationboard(stationID, limit, boardTime)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
