package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type StopHandler struct {
	stopService *services.StopService
}

func NewStopHandler(stopService *services.StopService) *StopHandler {
	return &StopHandler{stopService: stopService}
}

func (h *StopHandler) GetStop(c *gin.Context) {
	stopID := c.Param("id")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop id is required"})
		return
	}

	stop, err := h.stopService.GetByID(stopID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stop not found"})
		return
	}

	c.JSON(http.StatusOK, stop)
}

func (h *StopHandler) ListStops(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	stops, err := h.stopService.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list stops"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stops": stops,
		"count": len(stops),
	})
}

func (h *StopHandler) SearchStops(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	stops, err := h.stopService.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search stops"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stops": stops,
		"count": len(stops),
	})
}

func (h *StopHandler) FindNearby(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	radiusStr := c.DefaultQuery("radius", "500")

	if latStr == "" || lonStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lon are required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat"})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lon"})
		return
	}

	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil || radius < 0 || radius > 5000 {
		radius = 500
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	stops, err := h.stopService.FindNearby(lat, lon, radius, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find nearby stops"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stops": stops,
		"count": len(stops),
	})
}

func (h *StopHandler) GetDepartures(c *gin.Context) {
	stopID := c.Param("id")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	departures, err := h.stopService.GetDepartures(stopID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get departures"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"departures": departures,
		"count":      len(departures),
	})
}


