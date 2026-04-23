package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

type GoogleLoginRequest struct {
	Credential string `json:"credential" binding:"required"`
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "google credential is required"})
		return
	}

	session, err := h.authService.LoginWithGoogle(c.Request.Context(), req.Credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
		"user":       session.User,
		"provider":   session.Provider,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	session := getAuthSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":       session.User,
		"provider":   session.Provider,
		"expires_at": session.ExpiresAt,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetString("auth_token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	if err := h.authService.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func getAuthSession(c *gin.Context) *services.AuthSession {
	value, exists := c.Get("auth_session")
	if !exists {
		return nil
	}

	session, ok := value.(*services.AuthSession)
	if !ok {
		return nil
	}

	return session
}
