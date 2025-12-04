package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type JourneySessionHandler struct {
	sessionService      *services.JourneySessionService
	routeBoardingService *services.RouteBoardingService
}

func NewJourneySessionHandler(sessionService *services.JourneySessionService, routeBoardingService *services.RouteBoardingService) *JourneySessionHandler {
	return &JourneySessionHandler{
		sessionService:       sessionService,
		routeBoardingService: routeBoardingService,
	}
}

// CheckIn handles check-in request and generates QR code
func (h *JourneySessionHandler) CheckIn(c *gin.Context) {
	var req models.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	session, ticket, err := h.sessionService.CheckIn(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"qr_code": ticket.Code,
		"qr_ticket": ticket,
		"message": "Check-in successful. Show QR code to conductor.",
	})
}

// CheckOut handles check-out request
func (h *JourneySessionHandler) CheckOut(c *gin.Context) {
	var req models.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.SessionID == "" && req.QRCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id or qr_code is required"})
		return
	}

	// Use QR code if session ID not provided
	if req.SessionID == "" {
		req.SessionID = req.QRCode
	}
	if req.QRCode == "" {
		req.QRCode = req.SessionID
	}

	session, err := h.sessionService.CheckOut(req, h.routeBoardingService)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"message": "Check-out successful. Journey completed.",
		"fare":    session.TotalFare,
		"routes_used": session.RoutesUsed,
	})
}

// ValidateQR validates QR code and records boarding (for conductors/validators)
func (h *JourneySessionHandler) ValidateQR(c *gin.Context) {
	var req models.QRValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.QRCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr_code is required"})
		return
	}

	if req.RouteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}

	// Validate QR code
	ticket, err := h.sessionService.ValidateQRCode(req.QRCode, req.RouteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"error":   err.Error(),
			"qr_code": req.QRCode,
		})
		return
	}

	// Record boarding if route boarding service available
	if h.routeBoardingService != nil {
		// Get user's current location from request (if provided)
		var lat, lon float64
		if latStr := c.Query("latitude"); latStr != "" {
			if parsedLat, err := strconv.ParseFloat(latStr, 64); err == nil {
				lat = parsedLat
			}
		}
		if lonStr := c.Query("longitude"); lonStr != "" {
			if parsedLon, err := strconv.ParseFloat(lonStr, 64); err == nil {
				lon = parsedLon
			}
		}

		boardReq := models.BoardRouteRequest{
			QRCode:  req.QRCode,
			RouteID: req.RouteID,
			Latitude: lat,
			Longitude: lon,
		}

		boarding, err := h.routeBoardingService.BoardRoute(boardReq)
		if err != nil {
			// Log error but don't fail validation
			fmt.Printf("Warning: Failed to record boarding: %v\n", err)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"valid":     true,
				"qr_ticket": ticket,
				"boarding":  boarding,
				"message":   "QR code is valid. Boarding recorded.",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"qr_ticket": ticket,
		"message":   "QR code is valid",
	})
}

// GetActiveSessions returns active sessions for a user
func (h *JourneySessionHandler) GetActiveSessions(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	sessions, err := h.sessionService.GetActiveSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

