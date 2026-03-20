package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related requests.
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreateOrderRequest represents the create order request payload.
type CreateOrderRequest struct {
	AmountUSD float64 `json:"amount_usd" binding:"required"`
	PayType   string  `json:"pay_type" binding:"required"`
}

// CreateOrder handles creating a new payment order.
// POST /api/v1/payment/orders
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.paymentService.CreateOrder(c.Request.Context(), subject.UserID, service.CreateOrderRequest{
		AmountUSD: req.AmountUSD,
		PayType:   req.PayType,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// Notify handles 易支付 asynchronous callback notification.
// POST /api/v1/payment/notify
func (h *PaymentHandler) Notify(c *gin.Context) {
	// 易支付 sends callback params as form-urlencoded POST
	params := make(map[string]string)
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// Also try query params (some 易支付 implementations use GET)
	for key, values := range c.Request.URL.Query() {
		if _, exists := params[key]; !exists && len(values) > 0 {
			params[key] = values[0]
		}
	}

	// Parse form if not already parsed
	if len(params) == 0 {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.Form {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}
		}
	}

	err := h.paymentService.HandleNotify(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 易支付 expects "success" plain text response on successful processing
	c.String(200, "success")
}

// GetOrders returns the authenticated user's payment order history.
// GET /api/v1/payment/orders
func (h *PaymentHandler) GetOrders(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orders, err := h.paymentService.GetOrderHistory(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, orders)
}

// CancelOrder allows a user to cancel their own pending order.
// POST /api/v1/payment/orders/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	if err := h.paymentService.CancelOrder(c.Request.Context(), subject.UserID, orderID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Order cancelled"})
}

// RetryPayment regenerates a payment URL for an existing pending order.
// POST /api/v1/payment/orders/:id/pay
func (h *PaymentHandler) RetryPayment(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	result, err := h.paymentService.RetryPayment(c.Request.Context(), subject.UserID, orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetExchangeRate returns the current USD to RMB exchange rate.
// GET /api/v1/payment/exchange-rate
func (h *PaymentHandler) GetExchangeRate(c *gin.Context) {
	rate := h.paymentService.GetExchangeRate()
	response.Success(c, gin.H{"rate": rate})
}

// GetLimits returns the configured min/max topup amounts and preset amounts (USD).
// GET /api/v1/payment/limits
func (h *PaymentHandler) GetLimits(c *gin.Context) {
	minUSD, maxUSD, presetAmounts := h.paymentService.GetLimits()
	response.Success(c, gin.H{"min_usd": minUSD, "max_usd": maxUSD, "preset_amounts": presetAmounts})
}
