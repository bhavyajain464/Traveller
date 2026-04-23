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
	sessionService       *services.JourneySessionService
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
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}

	var req models.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.UserID = authUser.ID

	session, ticket, err := h.sessionService.CheckIn(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":   session,
		"qr_code":   ticket.Code,
		"qr_ticket": ticket,
		"message":   "Check-in successful. Show QR code to conductor.",
	})
}

// CheckOut handles check-out request
func (h *JourneySessionHandler) CheckOut(c *gin.Context) {
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}

	var req models.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var (
		session *models.JourneySession
		err     error
	)
	if req.SessionID == "" && req.QRCode == "" {
		session, err = h.sessionService.GetLatestActiveSession(authUser.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve active session"})
			return
		}
		if session == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no active session found for the authenticated user"})
			return
		}
	} else if req.SessionID != "" {
		session = userOwnsSession(c, h.sessionService, authUser.ID, req.SessionID)
	} else {
		session = userOwnsSessionByQRCode(c, h.sessionService, authUser.ID, req.QRCode)
	}
	if session == nil {
		return
	}
	req.SessionID = session.ID
	req.QRCode = session.QRCode

	session, err = h.sessionService.CheckOut(req, h.routeBoardingService)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":     session,
		"message":     "Check-out successful. Journey completed.",
		"fare":        session.TotalFare,
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
			QRCode:    req.QRCode,
			RouteID:   req.RouteID,
			Latitude:  lat,
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
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}

	if requestedUserID := c.Param("user_id"); requestedUserID != "" && requestedUserID != authUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot access sessions for another user"})
		return
	}

	sessions, err := h.sessionService.GetActiveSessions(authUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// ListMySessions returns recent sessions for the authenticated user.
func (h *JourneySessionHandler) ListMySessions(c *gin.Context) {
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}

	limit := 10
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	sessions, err := h.sessionService.ListSessions(authUser.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}
