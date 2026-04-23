package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

func requireAuthenticatedUser(c *gin.Context) *models.User {
	session := getAuthSession(c)
	if session == nil || session.User == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}

	return session.User
}

func userOwnsSession(c *gin.Context, sessionService *services.JourneySessionService, userID, sessionID string) *models.JourneySession {
	session, err := sessionService.GetSessionByID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return nil
	}
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to the authenticated user"})
		return nil
	}
	return session
}

func userOwnsSessionByQRCode(c *gin.Context, sessionService *services.JourneySessionService, userID, qrCode string) *models.JourneySession {
	session, err := sessionService.GetSessionByQRCode(qrCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return nil
	}
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to the authenticated user"})
		return nil
	}
	return session
}
