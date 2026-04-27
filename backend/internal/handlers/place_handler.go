package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type PlaceHandler struct {
	placeSearch *services.PlaceSearchService
}

func NewPlaceHandler(placeSearch *services.PlaceSearchService) *PlaceHandler {
	return &PlaceHandler{placeSearch: placeSearch}
}

func (h *PlaceHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	limit := 6
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 10 {
			limit = parsed
		}
	}

	var latPtr, lonPtr *float64
	if rawLat, rawLon := c.Query("lat"), c.Query("lon"); rawLat != "" && rawLon != "" {
		lat, latErr := strconv.ParseFloat(rawLat, 64)
		lon, lonErr := strconv.ParseFloat(rawLon, 64)
		if latErr == nil && lonErr == nil {
			latPtr = &lat
			lonPtr = &lon
		}
	}

	suggestions, err := h.placeSearch.Search(c.Request.Context(), services.PlaceSearchRequest{
		Query: query,
		Limit: limit,
		Lat:   latPtr,
		Lon:   lonPtr,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"count":       len(suggestions),
		"provider":    h.placeSearch.ProviderName(),
	})
}

func (h *PlaceHandler) Resolve(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'id' is required"})
		return
	}

	result, err := h.placeSearch.Resolve(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"place": result,
	})
}
