package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// EpayClient is the HTTP client for 易支付 payment gateway.
type EpayClient struct {
	cfg        config.EpayConfig
	httpClient *http.Client
}

// NewEpayClient creates a new EpayClient.
func NewEpayClient(cfg *config.Config) *EpayClient {
	return &EpayClient{
		cfg: cfg.Epay,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreatePayURL builds the 易支付 payment redirect URL for a given order.
func (c *EpayClient) CreatePayURL(orderNo string, amountRMB float64, payType string, itemName string) (string, error) {
	params := map[string]string{
		"pid":          fmt.Sprintf("%d", c.cfg.PID),
		"type":         payType,
		"out_trade_no": orderNo,
		"notify_url":   c.cfg.NotifyURL,
		"return_url":   c.cfg.ReturnURL,
		"name":         itemName,
		"money":        fmt.Sprintf("%.2f", amountRMB),
	}

	params["sign"] = c.sign(params)
	params["sign_type"] = "MD5"

	u, err := url.Parse(c.cfg.APIUrl + "/submit.php")
	if err != nil {
		return "", fmt.Errorf("parse epay api url: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// VerifyNotify verifies the MD5 signature from 易支付 callback notification.
// Returns true if the signature is valid.
func (c *EpayClient) VerifyNotify(params map[string]string) bool {
	receivedSign, ok := params["sign"]
	if !ok {
		return false
	}

	// Build params map without sign and sign_type for verification
	filtered := make(map[string]string, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if v == "" {
			continue
		}
		filtered[k] = v
	}

	expectedSign := c.sign(filtered)
	return receivedSign == expectedSign
}

// QueryOrderResponse represents the 易支付 order query result.
type QueryOrderResponse struct {
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	Type       string `json:"type"`
	Status     int    `json:"status"` // 1 = paid
	Money      string `json:"money"`
}

// QueryOrder queries an order status from 易支付.
func (c *EpayClient) QueryOrder(orderNo string) (*QueryOrderResponse, error) {
	params := map[string]string{
		"act":          "order",
		"pid":          fmt.Sprintf("%d", c.cfg.PID),
		"out_trade_no": orderNo,
	}
	params["sign"] = c.sign(params)
	params["sign_type"] = "MD5"

	u, err := url.Parse(c.cfg.APIUrl + "/api.php")
	if err != nil {
		return nil, fmt.Errorf("parse epay query url: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("epay query request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read epay query response: %w", err)
	}

	var result QueryOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse epay query response: %w", err)
	}

	return &result, nil
}

// GetExchangeRate returns the configured USD to RMB exchange rate.
func (c *EpayClient) GetExchangeRate() float64 {
	return c.cfg.USDToRMB
}

// GetRuntimeConfig returns an EpayConfig derived from DB settings (if enabled), falling back to the static config.
func (c *EpayClient) GetRuntimeConfig(settings *SystemSettings) config.EpayConfig {
	if settings != nil && settings.EpayEnabled {
		cfg := config.EpayConfig{
			Enabled:   true,
			APIUrl:    settings.EpayAPIUrl,
			PID:       settings.EpayPID,
			Key:       settings.EpayKey,
			NotifyURL: settings.EpayNotifyURL,
			ReturnURL: settings.EpayReturnURL,
			USDToRMB:  settings.EpayUSDToRMB,
		}
		return cfg
	}
	return c.cfg
}

// CreatePayURLWithConfig builds the payment URL using the given config (for runtime DB settings).
func (c *EpayClient) CreatePayURLWithConfig(cfg config.EpayConfig, orderNo string, amountRMB float64, payType string, itemName string) (string, error) {
	params := map[string]string{
		"pid":          fmt.Sprintf("%d", cfg.PID),
		"type":         payType,
		"out_trade_no": orderNo,
		"notify_url":   cfg.NotifyURL,
		"return_url":   cfg.ReturnURL,
		"name":         itemName,
		"money":        fmt.Sprintf("%.2f", amountRMB),
	}

	params["sign"] = c.signWithKey(params, cfg.Key)
	params["sign_type"] = "MD5"

	u, err := url.Parse(cfg.APIUrl + "/submit.php")
	if err != nil {
		return "", fmt.Errorf("parse epay api url: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// VerifyNotifyWithConfig verifies the MD5 signature using the given config.
func (c *EpayClient) VerifyNotifyWithConfig(cfg config.EpayConfig, params map[string]string) bool {
	receivedSign, ok := params["sign"]
	if !ok {
		return false
	}

	filtered := make(map[string]string, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if v == "" {
			continue
		}
		filtered[k] = v
	}

	expectedSign := c.signWithKey(filtered, cfg.Key)
	return receivedSign == expectedSign
}

// signWithKey generates MD5 signature using the provided merchant key.
func (c *EpayClient) signWithKey(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(params[k])
	}

	buf.WriteString(key)

	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}

// sign generates the MD5 signature per 易支付 spec.
// Steps: sort keys alphabetically, concatenate as key=value&, append merchant key, MD5 hash.
func (c *EpayClient) sign(params map[string]string) string {
	// Collect and sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(params[k])
	}

	// Append merchant key
	buf.WriteString(c.cfg.Key)

	// MD5 hash
	hash := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(hash[:])
}
