package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/services"
)

type DailyBillHandler struct {
	billService *services.DailyBillService
}

func NewDailyBillHandler(billService *services.DailyBillService) *DailyBillHandler {
	return &DailyBillHandler{billService: billService}
}

// GetDailyBill retrieves daily bill for a user
func (h *DailyBillHandler) GetDailyBill(c *gin.Context) {
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}
	userID := authUser.ID
	if requestedUserID := c.Param("user_id"); requestedUserID != "" && requestedUserID != authUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot access bills for another user"})
		return
	}

	dateStr := c.Query("date")
	var billDate time.Time
	var err error

	if dateStr == "" {
		billDate = time.Now()
	} else {
		billDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format. Use YYYY-MM-DD"})
			return
		}
	}

	bill, err := h.billService.GetDailyBill(userID, billDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily bill"})
		return
	}

	c.JSON(http.StatusOK, bill)
}

// GetPendingBills returns all pending bills for a user
func (h *DailyBillHandler) GetPendingBills(c *gin.Context) {
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}
	userID := authUser.ID
	if requestedUserID := c.Param("user_id"); requestedUserID != "" && requestedUserID != authUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot access bills for another user"})
		return
	}

	bills, err := h.billService.GetPendingBills(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pending bills"})
		return
	}

	totalAmount := 0.0
	for _, bill := range bills {
		totalAmount += bill.TotalFare
	}

	c.JSON(http.StatusOK, gin.H{
		"bills":        bills,
		"count":        len(bills),
		"total_amount": totalAmount,
	})
}

// PayBill marks a bill as paid
func (h *DailyBillHandler) PayBill(c *gin.Context) {
	authUser := requireAuthenticatedUser(c)
	if authUser == nil {
		return
	}

	billID := c.Param("bill_id")
	if billID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bill_id is required"})
		return
	}

	var req struct {
		PaymentID     string `json:"payment_id" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id and payment_method are required"})
		return
	}

	bill, err := h.billService.GetBillByID(billID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if bill.UserID != authUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot pay a bill for another user"})
		return
	}

	err = h.billService.MarkBillAsPaid(billID, req.PaymentID, req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark bill as paid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Bill marked as paid",
		"bill_id":        billID,
		"payment_id":     req.PaymentID,
		"payment_method": req.PaymentMethod,
	})
}

// GenerateDailyBills generates bills for all users for a specific date (admin endpoint)
func (h *DailyBillHandler) GenerateDailyBills(c *gin.Context) {
	if requireAuthenticatedUser(c) == nil {
		return
	}

	dateStr := c.Query("date")
	var billDate time.Time
	var err error

	if dateStr == "" {
		// Default to yesterday
		billDate = time.Now().AddDate(0, 0, -1)
	} else {
		billDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format. Use YYYY-MM-DD"})
			return
		}
	}

	err = h.billService.GenerateDailyBills(billDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate daily bills"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Daily bills generated successfully",
		"bill_date": billDate.Format("2006-01-02"),
	})
}

func (h *DailyBillHandler) calculateTotalAmount(bills []models.DailyBill) float64 {
	total := 0.0
	for _, bill := range bills {
		total += bill.TotalFare
	}
	return total
}
