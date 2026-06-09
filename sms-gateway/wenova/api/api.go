// Package api is a thin client for the Wenova SMS/OTP HTTP API.
//
// It sends messages via POST /sms/package and surfaces Wenova's 5-digit
// business codes (30xxx for SMS) as a typed APIError, rather than relying on
// the HTTP status code.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BaseURL is the default production endpoint.
const BaseURL = "https://apimicroservices.wenova.fun"

// sendPath is the SMS/OTP send endpoint, relative to the base URL.
const sendPath = "/sms/package"

// maxResponseBytes bounds how much of a response body we read, guarding against
// a misbehaving server or proxy streaming an unbounded payload.
const maxResponseBytes = 1 << 20 // 1 MiB

// userAgent identifies this client to the API.
const userAgent = "laogo-wenova/1.0"

// Client sends requests to the Wenova API.
type Client struct {
	// BaseURL overrides the API base URL. Empty means BaseURL.
	BaseURL string

	// HTTPClient performs requests. Nil means a client with a 30s timeout.
	HTTPClient *http.Client
}

// NewClient returns a Client targeting baseURL. An empty baseURL uses BaseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SendSMSParams are the parameters for SendSMS.
//
// Exactly one of Token or ScriptID must be supplied (code 30101 otherwise).
type SendSMSParams struct {
	// Header is the SMS header, e.g. "WNV-OTP" or "WNV-info".
	Header string
	// PhoneNumber is the recipient. Its format is validated by the server.
	PhoneNumber string
	// Message is the message content.
	Message string
	// Token is the API (script) token. Optional if ScriptID is set.
	Token string
	// ScriptID is the script ID. Optional if Token is set.
	ScriptID int
	// UsePackage, when true, deducts from the SMS package.
	UsePackage bool
}

// sendRequestData mirrors the JSON `data` envelope expected by the API.
type sendRequestData struct {
	Header      string `json:"header"`
	PhoneNumber string `json:"phoneNumber"`
	Message     string `json:"message"`
	Token       string `json:"token,omitempty"`
	ScriptID    int    `json:"scriptId,omitempty"`
	UsePackage  bool   `json:"usePackage"`
}

type sendRequest struct {
	Data sendRequestData `json:"data"`
}

// Response is the decoded API response body.
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// SendSMS sends an SMS/OTP message. It returns the decoded Response on success,
// or an *APIError carrying the Wenova code when the API reports a failure.
func (c *Client) SendSMS(ctx context.Context, p SendSMSParams) (*Response, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(sendRequest{Data: sendRequestData{
		Header:      p.Header,
		PhoneNumber: p.PhoneNumber,
		Message:     p.Message,
		Token:       p.Token,
		ScriptID:    p.ScriptID,
		UsePackage:  p.UsePackage,
	}})
	if err != nil {
		return nil, fmt.Errorf("wenova: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+sendPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("wenova: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("wenova: do request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("wenova: read response (http %d): %w", resp.StatusCode, err)
	}

	var out Response
	if err := json.Unmarshal(raw, &out); err != nil {
		// A non-JSON body (e.g. an HTML error page from a proxy) is most useful
		// reported verbatim, bounded to a short snippet.
		return nil, fmt.Errorf("wenova: decode response (http %d): %w: %s", resp.StatusCode, err, snippet(raw))
	}

	// Wenova signals success/failure via the business code, not HTTP status.
	if !isSuccess(out, resp.StatusCode) {
		return nil, &APIError{Code: out.Code, Message: out.Message, HTTPStatus: resp.StatusCode}
	}
	return &out, nil
}

// snippet returns a short, single-line preview of raw for error messages.
func snippet(raw []byte) string {
	const max = 256
	s := strings.TrimSpace(string(raw))
	if len(s) > max {
		s = s[:max] + "…"
	}
	return strings.ReplaceAll(s, "\n", " ")
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return BaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// isSuccess reports whether a response represents success. A non-2xx status or a
// 30xxx error code both indicate failure.
func isSuccess(r Response, status int) bool {
	if status < 200 || status >= 300 {
		return false
	}
	if r.Code != 0 && IsSMSCode(r.Code) {
		return false
	}
	return true
}
