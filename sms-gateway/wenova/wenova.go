// Package wenova is a high-level client for sending SMS/OTP messages through
// the Wenova API. It wraps the lower-level api package with sensible defaults.
package wenova

import (
	"context"

	"github.com/barluscuda/laogo/sms-gateway/wenova/api"
)

// Default SMS headers.
const (
	DefaultSMSInfoHeader = "WNV-info"
	DefaultSMSOTPHeader  = "WNV-OTP"
)

// Wenova sends SMS/OTP messages using a configured token.
type Wenova struct {
	token  string
	client *api.Client
}

// Option configures a Wenova during construction.
type Option func(*Wenova)

// WithBaseURL overrides the API base URL (default: production).
func WithBaseURL(baseURL string) Option {
	return func(w *Wenova) { w.client.BaseURL = baseURL }
}

// WithClient sets a custom api.Client (e.g. with a tuned http.Client).
func WithClient(c *api.Client) Option {
	return func(w *Wenova) { w.client = c }
}

// NewWenova returns a Wenova authenticated with the given script token.
func NewWenova(token string, options ...Option) *Wenova {
	w := &Wenova{
		token:  token,
		client: api.NewClient(""),
	}
	for _, opt := range options {
		opt(w)
	}
	return w
}

// Send sends message to phoneNumber under the given header, charging the cost
// to the wallet balance. The caller chooses the header (e.g. DefaultSMSInfoHeader,
// DefaultSMSOTPHeader, or a custom one). It returns an *api.APIError carrying
// the Wenova code on failure.
func (w *Wenova) Send(ctx context.Context, header, phoneNumber, message string) (*api.Response, error) {
	return w.client.SendSMS(ctx, api.SendSMSParams{
		Header:      header,
		PhoneNumber: phoneNumber,
		Message:     message,
		Token:       w.token,
		UsePackage:  false,
	})
}

// SendByPackage sends message to phoneNumber under the given header, deducting
// the cost from the SMS package. It returns an *api.APIError carrying the
// Wenova code on failure.
func (w *Wenova) SendByPackage(ctx context.Context, header, phoneNumber, message string) (*api.Response, error) {
	return w.client.SendSMS(ctx, api.SendSMSParams{
		Header:      header,
		PhoneNumber: phoneNumber,
		Message:     message,
		Token:       w.token,
		UsePackage:  true,
	})
}
