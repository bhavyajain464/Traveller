package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type RouteBoardingHandler struct {
	boardingService        *services.RouteBoardingService
	sessionService         *services.JourneySessionService
	autoAlightService      *services.AutoAlightService
	vehicleLocationService *services.VehicleLocationService
}

func NewRouteBoardingHandler(boardingService *services.RouteBoardingService, sessionService *services.JourneySessionService, autoAlightService *services.AutoAlightService, vehicleLocationService *services.VehicleLocationService) *RouteBoardingHandler {
	return &RouteBoardingHandler{
		boardingService:        boardingService,
		sessionService:         sessionService,
		autoAlightService:      autoAlightService,
		vehicleLocationService: vehicleLocationService,
	}
}

// BoardRoute records when user boards a route
func (h *RouteBoardingHandler) BoardRoute(c *gin.Context) {
	var req models.BoardRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.RouteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}

	boarding, err := h.boardingService.BoardRoute(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"boarding": boarding,
		"message":  "Route boarding recorded successfully",
	})
}

// AlightRoute records when user alights from a route
func (h *RouteBoardingHandler) AlightRoute(c *gin.Context) {
	var req models.AlightRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.BoardingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "boarding_id is required"})
		return
	}

	boarding, err := h.boardingService.AlightRoute(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"boarding": boarding,
		"message":  "Route alighting recorded successfully",
		"fare":     boarding.Fare,
	})
}

// GetSessionBoardings returns all route boardings for a session
func (h *RouteBoardingHandler) GetSessionBoardings(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	boardings, err := h.boardingService.GetBoardingsForSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get boardings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"boardings": boardings,
		"count":     len(boardings),
	})
}

// GetActiveBoarding returns the currently active boarding for a session
func (h *RouteBoardingHandler) GetActiveBoarding(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	boarding, err := h.boardingService.GetActiveBoarding(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active boarding"})
		return
	}

	if boarding == nil {
		c.JSON(http.StatusOK, gin.H{
			"active": false,
			"message": "No active boarding",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":   true,
		"boarding": boarding,
	})
}

// AutoDetectAndBoard automatically detects which vehicle user is on and records boarding
func (h *RouteBoardingHandler) AutoDetectAndBoard(c *gin.Context) {
	var req models.AutoBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Use QR code if session ID not provided
	sessionID := req.SessionID
	if sessionID == "" && req.QRCode != "" {
		// Get session by QR code
		session, err := h.sessionService.GetSessionByQRCode(req.QRCode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid QR code"})
			return
		}
		sessionID = session.ID
	}

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id or qr_code is required"})
		return
	}

	boarding, match, err := h.boardingService.AutoDetectAndBoard(sessionID, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Reset auto-alight counter when a new boarding is created
	// This ensures clean state for the new boarding
	if h.autoAlightService != nil {
		h.autoAlightService.ResetCounter(sessionID)
	}

	c.JSON(http.StatusOK, gin.H{
		"boarding":        boarding,
		"vehicle_match":   match,
		"message":         "Vehicle detected and boarding recorded automatically",
		"detected_route":  match.RouteID,
		"detected_mode":   getModeName(match.RouteType),
		"confidence":       match.Confidence,
		"distance_meters": match.Distance,
	})
}

// UpdateContinuousLocation handles continuous location updates and auto-detects alighting
func (h *RouteBoardingHandler) UpdateContinuousLocation(c *gin.Context) {
	var req models.ContinuousLocationUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Use QR code if session ID not provided
	sessionID := req.SessionID
	if sessionID == "" && req.QRCode != "" {
		session, err := h.sessionService.GetSessionByQRCode(req.QRCode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid QR code"})
			return
		}
		sessionID = session.ID
	}

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id or qr_code is required"})
		return
	}

	// Check for automatic alighting
	alightedBoarding, err := h.autoAlightService.CheckAndAlight(sessionID, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if alightedBoarding != nil {
		// User was automatically alighted
		c.JSON(http.StatusOK, models.ContinuousLocationResponse{
			OnVehicle:      false,
			Alighted:       true,
			AlightingStopID: alightedBoarding.AlightingStopID,
			Message:        "Automatically alighted at stop",
		})
		return
	}

	// Check if user is on a vehicle
	activeBoarding, err := h.boardingService.GetActiveBoarding(sessionID)
	if err != nil || activeBoarding == nil {
		c.JSON(http.StatusOK, models.ContinuousLocationResponse{
			OnVehicle: false,
			Alighted:  false,
			Message:   "No active boarding",
		})
		return
	}

	// Check if user is on the boarded vehicle
	if activeBoarding.VehicleID != nil {
		// First, always verify user is on the boarded vehicle (works immediately, even before 30 seconds)
		isOnVehicle, distance, verifyErr := h.vehicleLocationService.VerifyUserOnVehicle(*activeBoarding.VehicleID, req.Latitude, req.Longitude)
		if verifyErr == nil {
			if isOnVehicle {
				// User is on the vehicle (within tolerance)
				// After 30 seconds, try to confirm exact match for better accuracy
				exactMatch, err2 := h.vehicleLocationService.FindExactVehicleMatch(req.Latitude, req.Longitude, activeBoarding.VehicleID)
				if err2 == nil && exactMatch != nil {
					// Found exact match (after vehicles moved) - use this for more accurate distance
					c.JSON(http.StatusOK, models.ContinuousLocationResponse{
						OnVehicle:      true,
						VehicleID:      activeBoarding.VehicleID,
						RouteID:        &exactMatch.RouteID,
						DistanceMeters: exactMatch.Distance,
						Alighted:       false,
						Message:        fmt.Sprintf("✅ Confirmed: User is on vehicle %s (%.1fm away)", *activeBoarding.VehicleID, exactMatch.Distance),
					})
					return
				}
				
				// Before vehicles move or if exact match unavailable, use basic verification
				c.JSON(http.StatusOK, models.ContinuousLocationResponse{
					OnVehicle:      true,
					VehicleID:      activeBoarding.VehicleID,
					RouteID:        &activeBoarding.RouteID,
					DistanceMeters: distance,
					Alighted:       false,
					Message:        fmt.Sprintf("User is on vehicle (%.1fm away)", distance),
				})
				return
			} else {
				// User is not on the boarded vehicle - check if they're on a different vehicle (after movement)
				exactMatch, err2 := h.vehicleLocationService.FindExactVehicleMatch(req.Latitude, req.Longitude, activeBoarding.VehicleID)
				if err2 == nil && exactMatch != nil {
					// User is on a different vehicle than boarded
					matchedVehicleID := exactMatch.VehicleLocation.VehicleID
					c.JSON(http.StatusOK, models.ContinuousLocationResponse{
						OnVehicle:      true,
						VehicleID:      &matchedVehicleID,
						RouteID:        &exactMatch.RouteID,
						DistanceMeters: exactMatch.Distance,
						Alighted:       false,
						Message:        fmt.Sprintf("⚠️ Vehicle confirmed: User switched to vehicle %s (%.1fm away)", matchedVehicleID, exactMatch.Distance),
					})
					return
				}
				
				// User is not on any vehicle
				c.JSON(http.StatusOK, models.ContinuousLocationResponse{
					OnVehicle:      false,
					VehicleID:      nil,
					RouteID:        &activeBoarding.RouteID,
					DistanceMeters: distance,
					Alighted:       false,
					Message:        fmt.Sprintf("User is not on vehicle (%.1fm away)", distance),
				})
				return
			}
		}
	}
	
	// No vehicle ID or error
	c.JSON(http.StatusOK, models.ContinuousLocationResponse{
		OnVehicle: false,
		Alighted:  false,
		Message:   "No active boarding or vehicle tracking unavailable",
	})

	c.JSON(http.StatusOK, models.ContinuousLocationResponse{
		OnVehicle: true,
		RouteID:   &activeBoarding.RouteID,
		Alighted:  false,
		Message:   "Active boarding, vehicle tracking unavailable",
	})
}

// Helper function to get mode name
func getModeName(routeType int) string {
	switch routeType {
	case 1:
		return "Metro"
	case 3:
		return "Bus"
	case 2:
		return "Rail"
	default:
		return "Unknown"
	}
}

