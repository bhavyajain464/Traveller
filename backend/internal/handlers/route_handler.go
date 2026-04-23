package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type RouteHandler struct {
	routeService *services.RouteService
}

func NewRouteHandler(routeService *services.RouteService) *RouteHandler {
	return &RouteHandler{routeService: routeService}
}

func (h *RouteHandler) GetRoute(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route id is required"})
		return
	}

	route, err := h.routeService.GetByID(routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

func (h *RouteHandler) GetRouteDetail(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route id is required"})
		return
	}

	tripLimitStr := c.DefaultQuery("trip_limit", "12")
	tripLimit, err := strconv.Atoi(tripLimitStr)
	if err != nil || tripLimit < 1 || tripLimit > 100 {
		tripLimit = 12
	}

	detail, err := h.routeService.GetDetail(routeID, tripLimit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *RouteHandler) ListRoutes(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	agencyID := c.Query("agency_id")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	routes, err := h.routeService.List(limit, offset, agencyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list routes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routes": routes,
		"count":  len(routes),
	})
}

func (h *RouteHandler) SearchRoutes(c *gin.Context) {
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

	routes, err := h.routeService.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search routes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routes": routes,
		"count":  len(routes),
	})
}

func (h *RouteHandler) GetRouteStops(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route id is required"})
		return
	}

	stops, err := h.routeService.GetStops(routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get route stops"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stops": stops,
		"count": len(stops),
	})
}

func (h *RouteHandler) GetRouteTrips(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	trips, err := h.routeService.GetTrips(routeID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get route trips"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trips": trips,
		"count": len(trips),
	})
}
