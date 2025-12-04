package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/services"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	PhoneNumber string  `json:"phone_number" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Email       *string `json:"email,omitempty"`
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := h.userService.CreateUser(req.PhoneNumber, req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"message": "User created successfully",
	})
}

// GetUser retrieves a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteUserByPhone deletes a user by phone number
func (h *UserHandler) DeleteUserByPhone(c *gin.Context) {
	phoneNumber := c.Param("phone")
	if phoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone_number is required"})
		return
	}

	err := h.userService.DeleteUserByPhone(phoneNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
		"phone_number": phoneNumber,
	})
}

