package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminPaymentHandler handles admin payment order management
type AdminPaymentHandler struct {
	paymentService *service.PaymentService
}

// NewAdminPaymentHandler creates a new admin payment handler
func NewAdminPaymentHandler(paymentService *service.PaymentService) *AdminPaymentHandler {
	return &AdminPaymentHandler{
		paymentService: paymentService,
	}
}

// UpdateOrderStatusRequest represents the request body for updating order status
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ListOrders returns payment orders. Supports optional ?user_id= filter.
func (h *AdminPaymentHandler) ListOrders(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr != "" {
		uid, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "invalid user_id")
			return
		}
		orders, err := h.paymentService.ListAllOrdersByUser(c.Request.Context(), uid)
		if response.ErrorFrom(c, err) {
			return
		}
		response.Success(c, orders)
		return
	}

	orders, err := h.paymentService.ListAllOrders(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, orders)
}

// UpdateOrderStatus updates a payment order's status
func (h *AdminPaymentHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid order ID")
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.paymentService.UpdateOrderStatus(c.Request.Context(), id, req.Status); err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
