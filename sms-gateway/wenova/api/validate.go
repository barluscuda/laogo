package api

import "errors"

// Validation errors returned by SendSMSParams.Validate, before any request is
// made. They are distinct from the API's own 30xxx codes (see APIError).
var (
	ErrMissingHeader     = errors.New("wenova: header is required")
	ErrMissingPhone      = errors.New("wenova: phoneNumber is required")
	ErrMissingMessage    = errors.New("wenova: message is required")
	ErrMissingCredential = errors.New("wenova: either Token or ScriptID is required")
)

// Validate checks the parameters for the minimum a request needs: required
// fields and a credential. The phone number's format is intentionally not
// enforced here — the Wenova server is the source of truth for valid recipient
// numbers. It is called automatically by SendSMS, but is exported so callers can
// validate up front.
func (p SendSMSParams) Validate() error {
	if p.Header == "" {
		return ErrMissingHeader
	}
	if p.PhoneNumber == "" {
		return ErrMissingPhone
	}
	if p.Message == "" {
		return ErrMissingMessage
	}
	if p.Token == "" && p.ScriptID == 0 {
		return ErrMissingCredential
	}
	return nil
}
