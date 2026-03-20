package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
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
	cfg            *config.Config
	epayClient     *EpayClient
	entClient      *dbent.Client
	userRepo       UserRepository
	settingService *SettingService
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(cfg *config.Config, epayClient *EpayClient, entClient *dbent.Client, userRepo UserRepository, settingService *SettingService) *PaymentService {
	return &PaymentService{
		cfg:            cfg,
		epayClient:     epayClient,
		entClient:      entClient,
		userRepo:       userRepo,
		settingService: settingService,
	}
}

// isEnabled checks if payment is enabled (DB settings take priority over config).
func (s *PaymentService) isEnabled() bool {
	settings, err := s.settingService.GetAllSettings(context.Background())
	if err == nil {
		return settings.EpayEnabled
	}
	return s.cfg.Epay.Enabled
}

// getExchangeRateFromSettings returns the exchange rate from DB settings, falling back to config.
func (s *PaymentService) getExchangeRateFromSettings() float64 {
	settings, err := s.settingService.GetAllSettings(context.Background())
	if err == nil && settings.EpayUSDToRMB > 0 {
		return settings.EpayUSDToRMB
	}
	return s.cfg.Epay.USDToRMB
}

// getRuntimeConfig returns the runtime epay config from DB settings, falling back to config.
func (s *PaymentService) getRuntimeConfig() config.EpayConfig {
	settings, err := s.settingService.GetAllSettings(context.Background())
	if err == nil {
		return s.epayClient.GetRuntimeConfig(settings)
	}
	return s.cfg.Epay
}

// CreateOrder generates an order number, calculates RMB amount, and creates a 易支付 pay URL.
func (s *PaymentService) CreateOrder(ctx context.Context, userID int64, req CreateOrderRequest) (*CreateOrderResponse, error) {
	if !s.isEnabled() {
		return nil, ErrPaymentNotEnabled
	}

	if req.AmountUSD <= 0 {
		return nil, ErrInvalidAmount
	}

	// Validate min/max topup limits
	settings, _ := s.settingService.GetAllSettings(ctx)
	minUSD := 1.0
	maxUSD := 500.0
	if settings != nil {
		if settings.EpayMinTopupUSD > 0 {
			minUSD = settings.EpayMinTopupUSD
		}
		if settings.EpayMaxTopupUSD > 0 {
			maxUSD = settings.EpayMaxTopupUSD
		}
	}
	if req.AmountUSD < minUSD {
		return nil, infraerrors.BadRequest("AMOUNT_TOO_LOW", fmt.Sprintf("minimum topup is $%.2f", minUSD))
	}
	if req.AmountUSD > maxUSD {
		return nil, infraerrors.BadRequest("AMOUNT_TOO_HIGH", fmt.Sprintf("maximum topup is $%.2f", maxUSD))
	}

	// Generate order number: timestamp + random suffix
	orderNo := s.generateOrderNo()

	// Get runtime config from DB settings (falls back to config.yaml)
	runtimeCfg := s.getRuntimeConfig()

	// Calculate RMB amount from USD * exchange rate
	rate := runtimeCfg.USDToRMB
	amountRMB := req.AmountUSD * rate

	// Save order to database
	_, err := s.entClient.PaymentOrder.Create().
		SetUserID(userID).
		SetOrderNo(orderNo).
		SetAmountUsd(req.AmountUSD).
		SetAmountRmb(amountRMB).
		SetExchangeRate(rate).
		SetStatus("pending").
		SetPayType(req.PayType).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create payment order: %w", err)
	}

	logger.LegacyPrintf("service.payment", "Creating order %s for user %d: USD %.2f -> RMB %.2f (rate %.4f)",
		orderNo, userID, req.AmountUSD, amountRMB, rate)

	// Build pay URL using runtime config
	payURL, err := s.epayClient.CreatePayURLWithConfig(runtimeCfg, orderNo, amountRMB, req.PayType, "Account Top-Up")
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
	// Verify signature using runtime config
	runtimeCfg := s.getRuntimeConfig()
	if !s.epayClient.VerifyNotifyWithConfig(runtimeCfg, params) {
		return ErrInvalidSignature
	}

	tradeStatus := params["trade_status"]
	orderNo := params["out_trade_no"]
	tradeNo := params["trade_no"]

	logger.LegacyPrintf("service.payment", "Received notify for order %s: trade_no=%s, status=%s",
		orderNo, tradeNo, tradeStatus)

	if tradeStatus != "TRADE_SUCCESS" {
		return nil
	}

	// Look up order from database
	order, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.OrderNo(orderNo)).
		Only(ctx)
	if err != nil {
		return ErrOrderNotFound
	}

	if order.Status != "pending" {
		// Already processed, idempotent return
		return nil
	}

	// Update order status to paid
	now := time.Now()
	_, err = s.entClient.PaymentOrder.UpdateOne(order).
		SetTradeNo(tradeNo).
		SetStatus("paid").
		SetPaidAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update payment order status: %w", err)
	}

	// Credit user balance
	if err := s.userRepo.UpdateBalance(ctx, order.UserID, order.AmountUsd); err != nil {
		logger.LegacyPrintf("service.payment", "Failed to credit balance for user %d, order %s: %v",
			order.UserID, orderNo, err)
		return fmt.Errorf("credit user balance: %w", err)
	}

	logger.LegacyPrintf("service.payment", "Order %s paid: credited USD %.2f to user %d",
		orderNo, order.AmountUsd, order.UserID)

	return nil
}

// RetryPayment regenerates a payment URL for an existing pending order.
func (s *PaymentService) RetryPayment(ctx context.Context, userID int64, orderID int64) (*CreateOrderResponse, error) {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	if order.Status != "pending" {
		return nil, infraerrors.BadRequest("ORDER_NOT_PAYABLE", "only pending orders can be paid")
	}

	// Determine pay type, defaulting to alipay if nil
	payType := "alipay"
	if order.PayType != nil && *order.PayType != "" {
		payType = *order.PayType
	}

	runtimeCfg := s.getRuntimeConfig()
	payURL, err := s.epayClient.CreatePayURLWithConfig(runtimeCfg, order.OrderNo, order.AmountRmb, payType, "Account Top-Up")
	if err != nil {
		return nil, fmt.Errorf("create pay url: %w", err)
	}

	return &CreateOrderResponse{
		OrderNo: order.OrderNo,
		PayURL:  payURL,
	}, nil
}

// GetOrderHistory returns payment order history for a user.
func (s *PaymentService) GetOrderHistory(ctx context.Context, userID int64) ([]PaymentOrder, error) {
	orders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.UserID(userID)).
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query payment orders: %w", err)
	}

	result := make([]PaymentOrder, 0, len(orders))
	for _, o := range orders {
		result = append(result, PaymentOrder{
			ID:           o.ID,
			UserID:       o.UserID,
			OrderNo:      o.OrderNo,
			TradeNo:      o.TradeNo,
			AmountUSD:    o.AmountUsd,
			AmountRMB:    o.AmountRmb,
			ExchangeRate: o.ExchangeRate,
			Status:       o.Status,
			PayType:      o.PayType,
			PaidAt:       o.PaidAt,
			CreatedAt:    o.CreatedAt,
			UpdatedAt:    o.UpdatedAt,
		})
	}

	return result, nil
}

// CancelOrder allows a user to cancel their own pending order.
func (s *PaymentService) CancelOrder(ctx context.Context, userID int64, orderID int64) error {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return ErrOrderNotFound
	}
	if order.UserID != userID {
		return ErrOrderNotFound
	}
	if order.Status != "pending" {
		return infraerrors.BadRequest("ORDER_NOT_CANCELLABLE", "only pending orders can be cancelled")
	}
	_, err = s.entClient.PaymentOrder.UpdateOne(order).
		SetStatus("cancelled").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("cancel payment order: %w", err)
	}
	return nil
}

// ListAllOrdersByUser returns all payment orders for a specific user (admin use).
func (s *PaymentService) ListAllOrdersByUser(ctx context.Context, userID int64) ([]PaymentOrder, error) {
	orders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.UserID(userID)).
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query orders by user: %w", err)
	}
	result := make([]PaymentOrder, 0, len(orders))
	for _, o := range orders {
		result = append(result, PaymentOrder{
			ID:           o.ID,
			UserID:       o.UserID,
			OrderNo:      o.OrderNo,
			TradeNo:      o.TradeNo,
			AmountUSD:    o.AmountUsd,
			AmountRMB:    o.AmountRmb,
			ExchangeRate: o.ExchangeRate,
			Status:       o.Status,
			PayType:      o.PayType,
			PaidAt:       o.PaidAt,
			CreatedAt:    o.CreatedAt,
			UpdatedAt:    o.UpdatedAt,
		})
	}
	return result, nil
}

// GetLimits returns the configured min/max topup amounts and preset amounts (USD).
func (s *PaymentService) GetLimits() (minUSD float64, maxUSD float64, presetAmounts []float64) {
	minUSD = 1.0
	maxUSD = 500.0
	presetAmounts = []float64{5, 10, 20, 50, 100, 200}
	settings, err := s.settingService.GetAllSettings(context.Background())
	if err == nil {
		if settings.EpayMinTopupUSD > 0 {
			minUSD = settings.EpayMinTopupUSD
		}
		if settings.EpayMaxTopupUSD > 0 {
			maxUSD = settings.EpayMaxTopupUSD
		}
		if settings.EpayPresetAmounts != "" {
			var parsed []float64
			if json.Unmarshal([]byte(settings.EpayPresetAmounts), &parsed) == nil && len(parsed) > 0 {
				presetAmounts = parsed
			}
		}
	}
	return minUSD, maxUSD, presetAmounts
}

// GetExchangeRate returns the configured USD to RMB exchange rate (DB settings preferred).
func (s *PaymentService) GetExchangeRate() float64 {
	return s.getExchangeRateFromSettings()
}

// ListAllOrders returns all payment orders (admin use).
func (s *PaymentService) ListAllOrders(ctx context.Context) ([]PaymentOrder, error) {
	orders, err := s.entClient.PaymentOrder.Query().
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query all payment orders: %w", err)
	}

	result := make([]PaymentOrder, 0, len(orders))
	for _, o := range orders {
		result = append(result, PaymentOrder{
			ID:           o.ID,
			UserID:       o.UserID,
			OrderNo:      o.OrderNo,
			TradeNo:      o.TradeNo,
			AmountUSD:    o.AmountUsd,
			AmountRMB:    o.AmountRmb,
			ExchangeRate: o.ExchangeRate,
			Status:       o.Status,
			PayType:      o.PayType,
			PaidAt:       o.PaidAt,
			CreatedAt:    o.CreatedAt,
			UpdatedAt:    o.UpdatedAt,
		})
	}

	return result, nil
}

// UpdateOrderStatus updates a payment order's status (admin use).
// If transitioning from "pending" to "paid", credits the user's balance.
func (s *PaymentService) UpdateOrderStatus(ctx context.Context, orderID int64, newStatus string) error {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return ErrOrderNotFound
	}

	oldStatus := order.Status

	// If already in the target status, nothing to do
	if oldStatus == newStatus {
		return nil
	}

	update := s.entClient.PaymentOrder.UpdateOne(order).
		SetStatus(newStatus)

	if newStatus == "paid" {
		now := time.Now()
		update = update.SetPaidAt(now)
	}

	if _, err := update.Save(ctx); err != nil {
		return fmt.Errorf("update payment order status: %w", err)
	}

	// Credit user balance only when transitioning from pending to paid
	if newStatus == "paid" && oldStatus == "pending" {
		if err := s.userRepo.UpdateBalance(ctx, order.UserID, order.AmountUsd); err != nil {
			logger.LegacyPrintf("service.payment", "Failed to credit balance for user %d, order %d: %v",
				order.UserID, orderID, err)
			return fmt.Errorf("credit user balance: %w", err)
		}
		logger.LegacyPrintf("service.payment", "Admin updated order %d to paid: credited USD %.2f to user %d",
			orderID, order.AmountUsd, order.UserID)
	}

	return nil
}

// generateOrderNo generates a unique order number using timestamp + random suffix.
func (s *PaymentService) generateOrderNo() string {
	now := time.Now()
	ts := now.Format("20060102150405")
	suffix := fmt.Sprintf("%06d", rand.Intn(1000000))
	return fmt.Sprintf("PAY%s%s", ts, suffix)
}
