package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type RouteBoardingHandler struct {
	boardingService *services.RouteBoardingService
}

func NewRouteBoardingHandler(boardingService *services.RouteBoardingService) *RouteBoardingHandler {
	return &RouteBoardingHandler{boardingService: boardingService}
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


