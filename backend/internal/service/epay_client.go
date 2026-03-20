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
