package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type VehicleLocationHandler struct {
	vehicleLocationService *services.VehicleLocationService
}

func NewVehicleLocationHandler(vehicleLocationService *services.VehicleLocationService) *VehicleLocationHandler {
	return &VehicleLocationHandler{
		vehicleLocationService: vehicleLocationService,
	}
}

// AddMockVehicle adds a mock vehicle at a specific location
func (h *VehicleLocationHandler) AddMockVehicle(c *gin.Context) {
	var req models.MockVehicleSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.RouteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}

	vehicleID := h.vehicleLocationService.AddMockVehicleWithTrip(req.RouteID, req.Latitude, req.Longitude, req.TripID, req.StartMovingImmediately)
	
	c.JSON(http.StatusOK, gin.H{
		"vehicle_id": vehicleID,
		"route_id":   req.RouteID,
		"latitude":   req.Latitude,
		"longitude":  req.Longitude,
		"start_moving_immediately": req.StartMovingImmediately,
		"message":    "Mock vehicle added successfully",
	})
}

// GetVehicleLocation gets the current location of a specific vehicle
func (h *VehicleLocationHandler) GetVehicleLocation(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")
	if vehicleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle_id is required"})
		return
	}

	location, err := h.vehicleLocationService.GetVehicleLocation(vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vehicle_location": location,
	})
}

