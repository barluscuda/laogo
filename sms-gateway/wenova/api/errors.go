package api

import (
	"errors"
	"fmt"
)

// Wenova SMS business codes (30xxx). Use these to branch on the failure reason
// returned in APIError.Code.
const (
	CodeTokenOrScriptIDRequired   = 30101 // Either token or scriptId is required
	CodeLinksNotAllowed           = 30102 // Links are not allowed in SMS messages
	CodePackageQuotaInsufficient  = 30105 // SMS package quota is insufficient
	CodeWalletBalanceInsufficient = 30106 // Wallet balance is insufficient for SMS segments
	CodeTokenNotFound             = 30307 // SMS API token not found
	CodeScriptIDNotFound          = 30308 // SMS scriptId not found
	CodeWalletNotFound            = 30302 // Wallet not found for user
	CodePackageNotFound           = 30303 // SMS package not found
	CodeRateLimitExceeded         = 30501 // SMS rate limit exceeded
	CodeGatewayError              = 30901 // SMS gateway returned an error
)

// IsSMSCode reports whether code is in the SMS range (30000–30999).
func IsSMSCode(code int) bool {
	return code >= 30000 && code <= 30999
}

// Sentinel errors, one per Wenova code, for use with errors.Is:
//
//	if errors.Is(err, api.ErrRateLimitExceeded) {
//		// back off and retry
//	}
//
// Matching is by code, so these work even when the returned error wraps extra
// context (a different Message or HTTPStatus still matches).
var (
	ErrTokenOrScriptIDRequired   = &APIError{Code: CodeTokenOrScriptIDRequired, Message: "either token or scriptId is required"}
	ErrLinksNotAllowed           = &APIError{Code: CodeLinksNotAllowed, Message: "links are not allowed in SMS messages"}
	ErrPackageQuotaInsufficient  = &APIError{Code: CodePackageQuotaInsufficient, Message: "SMS package quota is insufficient"}
	ErrWalletBalanceInsufficient = &APIError{Code: CodeWalletBalanceInsufficient, Message: "wallet balance is insufficient for SMS segments"}
	ErrTokenNotFound             = &APIError{Code: CodeTokenNotFound, Message: "SMS API token not found"}
	ErrScriptIDNotFound          = &APIError{Code: CodeScriptIDNotFound, Message: "SMS scriptId not found"}
	ErrWalletNotFound            = &APIError{Code: CodeWalletNotFound, Message: "wallet not found for user"}
	ErrPackageNotFound           = &APIError{Code: CodePackageNotFound, Message: "SMS package not found"}
	ErrRateLimitExceeded         = &APIError{Code: CodeRateLimitExceeded, Message: "SMS rate limit exceeded"}
	ErrGatewayError              = &APIError{Code: CodeGatewayError, Message: "SMS gateway returned an error"}
)

// APIError is returned when the Wenova API reports a non-success response. The
// Code is Wenova's 5-digit business id (e.g. 30101), not the HTTP status.
type APIError struct {
	Code       int
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("wenova: code %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("wenova: code %d (http %d)", e.Code, e.HTTPStatus)
}

// Is reports whether target is an *APIError with the same code, letting callers
// match against the sentinels with errors.Is regardless of Message/HTTPStatus.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	return ok && t.Code == e.Code
}

// Category groups related codes so callers can react without memorizing every
// numeric code. Use APIError.Category or the package-level CategoryOf helper.
type Category int

const (
	// CategoryUnknown is any error that is not a recognized APIError.
	CategoryUnknown Category = iota
	// CategoryInvalidRequest: the request is malformed; fix it and do not retry
	// as-is (e.g. missing credential, links in message).
	CategoryInvalidRequest
	// CategoryAuth: the token or scriptId was rejected.
	CategoryAuth
	// CategoryInsufficientFunds: SMS package quota or wallet balance is too low.
	CategoryInsufficientFunds
	// CategoryNotFound: a referenced resource (wallet, package) does not exist.
	CategoryNotFound
	// CategoryRateLimited: too many requests; retry later with backoff.
	CategoryRateLimited
	// CategoryGateway: the upstream SMS gateway failed; may be transient.
	CategoryGateway
)

// Category classifies the error by its Wenova code.
func (e *APIError) Category() Category {
	switch e.Code {
	case CodeTokenOrScriptIDRequired, CodeLinksNotAllowed:
		return CategoryInvalidRequest
	case CodeTokenNotFound, CodeScriptIDNotFound:
		return CategoryAuth
	case CodePackageQuotaInsufficient, CodeWalletBalanceInsufficient:
		return CategoryInsufficientFunds
	case CodeWalletNotFound, CodePackageNotFound:
		return CategoryNotFound
	case CodeRateLimitExceeded:
		return CategoryRateLimited
	case CodeGatewayError:
		return CategoryGateway
	default:
		return CategoryUnknown
	}
}

// Retryable reports whether retrying the same request later might succeed.
func (e *APIError) Retryable() bool {
	switch e.Category() {
	case CategoryRateLimited, CategoryGateway:
		return true
	default:
		return false
	}
}

// AsAPIError extracts an *APIError from err (unwrapping as needed). The bool is
// false for nil errors and for errors that are not API errors (e.g. validation
// or network failures).
func AsAPIError(err error) (*APIError, bool) {
	var e *APIError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// CategoryOf returns the Category of err, or CategoryUnknown if err is not an
// *APIError. It is the easy entry point for handling: switch on the result.
func CategoryOf(err error) Category {
	if e, ok := AsAPIError(err); ok {
		return e.Category()
	}
	return CategoryUnknown
}

// CodeOf returns the Wenova code carried by err, or 0 if err is not an *APIError.
func CodeOf(err error) int {
	if e, ok := AsAPIError(err); ok {
		return e.Code
	}
	return 0
}

// IsRetryable reports whether err is an API error that may succeed on retry.
func IsRetryable(err error) bool {
	e, ok := AsAPIError(err)
	return ok && e.Retryable()
}
