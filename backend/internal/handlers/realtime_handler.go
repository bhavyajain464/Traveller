package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type RealtimeHandler struct {
	realtimeService *services.RealtimeService
}

func NewRealtimeHandler(realtimeService *services.RealtimeService) *RealtimeHandler {
	return &RealtimeHandler{realtimeService: realtimeService}
}

func (h *RealtimeHandler) GetStopRealtime(c *gin.Context) {
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

	arrivals, err := h.realtimeService.GetStopArrivals(stopID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get real-time arrivals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"arrivals": arrivals,
		"count":    len(arrivals),
	})
}

func (h *RealtimeHandler) GetTripRealtime(c *gin.Context) {
	tripID := c.Param("id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trip id is required"})
		return
	}

	update, err := h.realtimeService.GetTripUpdate(tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trip update"})
		return
	}

	if update == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trip update not found"})
		return
	}

	c.JSON(http.StatusOK, update)
}

