package api

import (
	"errors"
	"fmt"
	"testing"
)

func TestCategoryOf(t *testing.T) {
	cases := []struct {
		code int
		want Category
	}{
		{CodeTokenOrScriptIDRequired, CategoryInvalidRequest},
		{CodeLinksNotAllowed, CategoryInvalidRequest},
		{CodeTokenNotFound, CategoryAuth},
		{CodePackageQuotaInsufficient, CategoryInsufficientFunds},
		{CodeWalletBalanceInsufficient, CategoryInsufficientFunds},
		{CodeWalletNotFound, CategoryNotFound},
		{CodeRateLimitExceeded, CategoryRateLimited},
		{CodeGatewayError, CategoryGateway},
		{39999, CategoryUnknown},
	}
	for _, tc := range cases {
		err := error(&APIError{Code: tc.code})
		if got := CategoryOf(err); got != tc.want {
			t.Errorf("CategoryOf(code %d) = %d, want %d", tc.code, got, tc.want)
		}
	}
}

func TestCategoryOf_NonAPIError(t *testing.T) {
	if got := CategoryOf(ErrMissingPhone); got != CategoryUnknown {
		t.Errorf("CategoryOf(validation err) = %d, want CategoryUnknown", got)
	}
	if got := CodeOf(ErrMissingPhone); got != 0 {
		t.Errorf("CodeOf(validation err) = %d, want 0", got)
	}
}

func TestIsRetryable(t *testing.T) {
	retry := []int{CodeRateLimitExceeded, CodeGatewayError}
	for _, c := range retry {
		if !IsRetryable(&APIError{Code: c}) {
			t.Errorf("IsRetryable(code %d) = false, want true", c)
		}
	}
	if IsRetryable(&APIError{Code: CodePackageQuotaInsufficient}) {
		t.Error("IsRetryable(quota) = true, want false")
	}
	if IsRetryable(ErrMissingHeader) {
		t.Error("IsRetryable(validation err) = true, want false")
	}
}

func TestErrorsIs_Sentinels(t *testing.T) {
	// A fresh error from the API (different Message/HTTPStatus) still matches the
	// sentinel by code, both directly and when wrapped.
	got := &APIError{Code: CodeRateLimitExceeded, Message: "too many", HTTPStatus: 429}
	if !errors.Is(got, ErrRateLimitExceeded) {
		t.Error("errors.Is(got, ErrRateLimitExceeded) = false, want true")
	}
	if !errors.Is(fmt.Errorf("send: %w", got), ErrRateLimitExceeded) {
		t.Error("errors.Is(wrapped, ErrRateLimitExceeded) = false, want true")
	}
	if errors.Is(got, ErrWalletBalanceInsufficient) {
		t.Error("errors.Is matched the wrong sentinel")
	}
}

func TestAsAPIError_Wrapped(t *testing.T) {
	base := &APIError{Code: CodeRateLimitExceeded, Message: "slow down"}
	wrapped := fmt.Errorf("send failed: %w", base)
	e, ok := AsAPIError(wrapped)
	if !ok || e.Code != CodeRateLimitExceeded {
		t.Fatalf("AsAPIError(wrapped) = %v, %v", e, ok)
	}
	if !errors.Is(wrapped, error(base)) {
		t.Error("errors.Is should find the wrapped APIError")
	}
}
