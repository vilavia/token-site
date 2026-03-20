package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrPaymentNotEnabled = infraerrors.BadRequest("PAYMENT_NOT_ENABLED", "payment is not enabled")
	ErrInvalidAmount     = infraerrors.BadRequest("INVALID_AMOUNT", "invalid payment amount")
	ErrOrderNotFound     = infraerrors.NotFound("ORDER_NOT_FOUND", "payment order not found")
	ErrInvalidSignature  = infraerrors.BadRequest("INVALID_SIGNATURE", "invalid payment callback signature")
)

// PaymentOrder represents a payment order returned to callers.
type PaymentOrder struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	OrderNo      string     `json:"order_no"`
	TradeNo      *string    `json:"trade_no,omitempty"`
	AmountUSD    float64    `json:"amount_usd"`
	AmountRMB    float64    `json:"amount_rmb"`
	ExchangeRate float64    `json:"exchange_rate"`
	Status       string     `json:"status"`
	PayType      *string    `json:"pay_type,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateOrderRequest is the request for creating a payment order.
type CreateOrderRequest struct {
	AmountUSD float64 `json:"amount_usd" binding:"required"`
	PayType   string  `json:"pay_type" binding:"required"` // alipay or wxpay
}

// CreateOrderResponse is the response for a newly created payment order.
type CreateOrderResponse struct {
	OrderNo string `json:"order_no"`
	PayURL  string `json:"pay_url"`
}

// PaymentService handles payment business logic.
type PaymentService struct {
	cfg        *config.Config
	epayClient *EpayClient
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(cfg *config.Config, epayClient *EpayClient) *PaymentService {
	return &PaymentService{
		cfg:        cfg,
		epayClient: epayClient,
	}
}

// CreateOrder generates an order number, calculates RMB amount, and creates a 易支付 pay URL.
func (s *PaymentService) CreateOrder(ctx context.Context, userID int64, req CreateOrderRequest) (*CreateOrderResponse, error) {
	if !s.cfg.Epay.Enabled {
		return nil, ErrPaymentNotEnabled
	}

	if req.AmountUSD <= 0 {
		return nil, ErrInvalidAmount
	}

	// Generate order number: timestamp + random suffix
	orderNo := s.generateOrderNo()

	// Calculate RMB amount from USD * exchange rate
	rate := s.epayClient.GetExchangeRate()
	amountRMB := req.AmountUSD * rate

	// TODO: Save order to database once Ent code is generated
	// order, err := s.entClient.PaymentOrder.Create().
	// 	SetUserID(userID).
	// 	SetOrderNo(orderNo).
	// 	SetAmountUsd(req.AmountUSD).
	// 	SetAmountRmb(amountRMB).
	// 	SetExchangeRate(rate).
	// 	SetStatus("pending").
	// 	SetPayType(req.PayType).
	// 	Save(ctx)

	logger.LegacyPrintf("service.payment", "Creating order %s for user %d: USD %.2f -> RMB %.2f (rate %.4f)",
		orderNo, userID, req.AmountUSD, amountRMB, rate)

	// Build pay URL
	payURL, err := s.epayClient.CreatePayURL(orderNo, amountRMB, req.PayType, "Account Top-Up")
	if err != nil {
		return nil, fmt.Errorf("create pay url: %w", err)
	}

	return &CreateOrderResponse{
		OrderNo: orderNo,
		PayURL:  payURL,
	}, nil
}

// HandleNotify processes 易支付 asynchronous callback notifications.
func (s *PaymentService) HandleNotify(ctx context.Context, params map[string]string) error {
	// Verify signature
	if !s.epayClient.VerifyNotify(params) {
		return ErrInvalidSignature
	}

	tradeStatus := params["trade_status"]
	orderNo := params["out_trade_no"]
	tradeNo := params["trade_no"]

	logger.LegacyPrintf("service.payment", "Received notify for order %s: trade_no=%s, status=%s",
		orderNo, tradeNo, tradeStatus)

	if tradeStatus != "TRADE_SUCCESS" {
		// Not a successful payment, log and skip
		return nil
	}

	// TODO: Look up order from database and update status
	// order, err := s.entClient.PaymentOrder.Query().
	// 	Where(paymentorder.OrderNo(orderNo)).
	// 	Only(ctx)
	// if err != nil {
	// 	return ErrOrderNotFound
	// }
	//
	// if order.Status != "pending" {
	// 	// Already processed, idempotent return
	// 	return nil
	// }
	//
	// now := time.Now()
	// _, err = s.entClient.PaymentOrder.UpdateOne(order).
	// 	SetTradeNo(tradeNo).
	// 	SetStatus("paid").
	// 	SetPaidAt(now).
	// 	Save(ctx)
	//
	// // TODO: Credit user balance
	// _, err = s.entClient.User.UpdateOneID(order.UserID).
	// 	AddBalance(order.AmountUSD).
	// 	Save(ctx)

	return nil
}

// GetOrderHistory returns payment order history for a user.
func (s *PaymentService) GetOrderHistory(ctx context.Context, userID int64) ([]PaymentOrder, error) {
	// TODO: Query orders from database once Ent code is generated
	// orders, err := s.entClient.PaymentOrder.Query().
	// 	Where(paymentorder.UserID(userID)).
	// 	Order(ent.Desc(paymentorder.FieldCreatedAt)).
	// 	All(ctx)

	return []PaymentOrder{}, nil
}

// GetExchangeRate returns the configured USD to RMB exchange rate.
func (s *PaymentService) GetExchangeRate() float64 {
	return s.cfg.Epay.USDToRMB
}

// generateOrderNo generates a unique order number using timestamp + random suffix.
func (s *PaymentService) generateOrderNo() string {
	now := time.Now()
	ts := now.Format("20060102150405")
	suffix := fmt.Sprintf("%06d", rand.Intn(1000000))
	return fmt.Sprintf("PAY%s%s", ts, suffix)
}
