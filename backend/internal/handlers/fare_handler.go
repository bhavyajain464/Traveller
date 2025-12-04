package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type FareHandler struct {
	fareService *services.FareService
}

func NewFareHandler(fareService *services.FareService) *FareHandler {
	return &FareHandler{fareService: fareService}
}

func (h *FareHandler) CalculateFare(c *gin.Context) {
	// Get journey details from query params
	fromStopID := c.Query("from_stop_id")
	toStopID := c.Query("to_stop_id")
	routeID := c.Query("route_id")

	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}

	if fromStopID == "" || toStopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_stop_id and to_stop_id are required"})
		return
	}

	agencyID := h.fareService.GetAgencyIDFromRoute(routeID)
	if agencyID == "" {
		agencyID = "DIMTS" // Default to Delhi bus if not found
	}
	rules := h.fareService.GetFareRulesForAgency(agencyID)
	fare, err := h.fareService.GetRouteFare(routeID, fromStopID, toStopID, rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate fare"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"route_id":    routeID,
		"from_stop":  fromStopID,
		"to_stop":    toStopID,
		"fare":       fare,
		"currency":   "INR",
		"fare_rules": rules,
	})
}

func (h *FareHandler) GetRouteFare(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route id is required"})
		return
	}

	fromStopID := c.Query("from_stop_id")
	toStopID := c.Query("to_stop_id")

	agencyID := h.fareService.GetAgencyIDFromRoute(routeID)
	if agencyID == "" {
		agencyID = "DIMTS" // Default to Delhi bus if not found
	}
	rules := h.fareService.GetFareRulesForAgency(agencyID)
	fare, err := h.fareService.GetRouteFare(routeID, fromStopID, toStopID, rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get fare"})
		return
	}

	response := gin.H{
		"route_id":  routeID,
		"base_fare": rules.BaseFare,
		"fare_per_km": rules.FarePerKm,
		"currency": "INR",
	}

	if fromStopID != "" && toStopID != "" {
		response["from_stop"] = fromStopID
		response["to_stop"] = toStopID
		response["fare"] = fare
	}

	c.JSON(http.StatusOK, response)
}

