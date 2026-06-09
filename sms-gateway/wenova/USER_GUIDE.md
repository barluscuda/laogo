# Wenova SMS Gateway User Guide

This guide is for application developers who want to send SMS or OTP messages
with the `laogo` Wenova client.

Use the high-level `wenova` package for most applications. Use the lower-level
`wenova/api` package only when you need direct access to request parameters such
as `ScriptID`.

## Install

```bash
go get github.com/barluscuda/laogo/sms-gateway/wenova
```

Import the package:

```go
import "github.com/barluscuda/laogo/sms-gateway/wenova"
```

The module currently requires Go 1.26.3 or newer.

## What You Need Before Sending

You need these values from Wenova:

| Value | Description | Where it is used |
| ----- | ----------- | ---------------- |
| Script token | API token for your SMS script | `wenova.NewWenova(token)` |
| SMS header | Sender header shown to the recipient | `header` argument |
| Recipient phone number | Destination phone number | `phoneNumber` argument |
| Message | SMS or OTP text | `message` argument |

The client checks that these fields are not empty before making an HTTP request.
Phone-number format is left to Wenova because the server is the source of truth.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/barluscuda/laogo/sms-gateway/wenova"
)

func main() {
	client := wenova.NewWenova("YOUR_SCRIPT_TOKEN")

	resp, err := client.Send(
		context.Background(),
		wenova.DefaultSMSOTPHeader,
		"2055123456",
		"Your verification code is 123456",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("SMS sent: code=%d success=%v message=%s\n", resp.Code, resp.Success, resp.Message)
}
```

## Choose Wallet or Package

Wenova supports two charging modes. The method names make the choice explicit.

| Method | Charging mode | Use when |
| ------ | ------------- | -------- |
| `Send` | Wallet balance | You want each SMS charged from wallet balance |
| `SendByPackage` | SMS package quota | You bought an SMS package and want to use that quota |

Wallet example:

```go
resp, err := client.Send(ctx, wenova.DefaultSMSInfoHeader, "2055123456", "Your order is ready")
```

Package example:

```go
resp, err := client.SendByPackage(ctx, wenova.DefaultSMSOTPHeader, "2055123456", "Your code is 123456")
```

Both methods return `(*api.Response, error)`.

## SMS Headers

Two default headers are available:

| Constant | Value | Typical use |
| -------- | ----- | ----------- |
| `wenova.DefaultSMSInfoHeader` | `WNV-info` | General notification messages |
| `wenova.DefaultSMSOTPHeader` | `WNV-OTP` | OTP or verification messages |

You can also pass your own header if your Wenova account supports it:

```go
resp, err := client.Send(ctx, "MY-APP", "2055123456", "Welcome to My App")
```

## Recommended Error Handling

Handle the common business errors first, then fall back to error categories.

```go
package main

import (
	"context"
	"errors"
	"log"

	"github.com/barluscuda/laogo/sms-gateway/wenova"
	"github.com/barluscuda/laogo/sms-gateway/wenova/api"
)

func sendOTP(ctx context.Context, token, phone, code string) error {
	client := wenova.NewWenova(token)
	_, err := client.Send(ctx, wenova.DefaultSMSOTPHeader, phone, "Your code is "+code)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, api.ErrRateLimitExceeded):
		log.Println("too many SMS requests; retry later with backoff")
	case errors.Is(err, api.ErrWalletBalanceInsufficient):
		log.Println("wallet balance is too low")
	case errors.Is(err, api.ErrPackageQuotaInsufficient):
		log.Println("SMS package quota is empty")
	case errors.Is(err, api.ErrTokenNotFound), errors.Is(err, api.ErrScriptIDNotFound):
		log.Println("Wenova token or script ID is invalid")
	default:
		log.Printf("send failed: %v", err)
	}

	return err
}
```

Category-based handling is useful when you do not care about every exact code:

```go
switch api.CategoryOf(err) {
case api.CategoryRateLimited, api.CategoryGateway:
	// Retry later with backoff.
case api.CategoryInsufficientFunds:
	// Top up wallet or SMS package.
case api.CategoryAuth:
	// Check token or scriptId.
case api.CategoryInvalidRequest:
	// Fix request data before retrying.
}
```

## Common Error Codes

| Code | Error | What to do |
| ---- | ----- | ---------- |
| `30101` | `api.ErrTokenOrScriptIDRequired` | Provide a token or script ID |
| `30102` | `api.ErrLinksNotAllowed` | Remove links from the SMS message |
| `30105` | `api.ErrPackageQuotaInsufficient` | Buy or renew an SMS package |
| `30106` | `api.ErrWalletBalanceInsufficient` | Top up wallet balance |
| `30302` | `api.ErrWalletNotFound` | Check the Wenova account wallet |
| `30303` | `api.ErrPackageNotFound` | Check the SMS package setup |
| `30307` | `api.ErrTokenNotFound` | Check the API token |
| `30308` | `api.ErrScriptIDNotFound` | Check the script ID |
| `30501` | `api.ErrRateLimitExceeded` | Retry later with backoff |
| `30901` | `api.ErrGatewayError` | Retry later or contact Wenova support |

## Use a Timeout

`api.NewClient` uses a 30 second timeout by default. For production systems, use
a timeout that matches your request path.

```go
package main

import (
	"net/http"
	"time"

	"github.com/barluscuda/laogo/sms-gateway/wenova"
	"github.com/barluscuda/laogo/sms-gateway/wenova/api"
)

func newClient(token string) *wenova.Wenova {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiClient := &api.Client{HTTPClient: httpClient}

	return wenova.NewWenova(token, wenova.WithClient(apiClient))
}
```

## Validate Before Sending

The low-level API lets you validate request parameters before a send attempt.

```go
params := api.SendSMSParams{
	Header:      wenova.DefaultSMSOTPHeader,
	PhoneNumber: "2055123456",
	Message:     "Your code is 123456",
	Token:       "YOUR_SCRIPT_TOKEN",
}

if err := params.Validate(); err != nil {
	// Fix the request before calling SendSMS.
}
```

Validation checks:

| Missing field | Returned error |
| ------------- | -------------- |
| Header | `api.ErrMissingHeader` |
| Phone number | `api.ErrMissingPhone` |
| Message | `api.ErrMissingMessage` |
| Token and script ID | `api.ErrMissingCredential` |

## Low-Level API

Use `wenova/api` directly when you need `ScriptID` instead of a token or when you
want to build custom wrappers.

```go
package main

import (
	"context"
	"log"

	"github.com/barluscuda/laogo/sms-gateway/wenova"
	"github.com/barluscuda/laogo/sms-gateway/wenova/api"
)

func main() {
	client := api.NewClient("")

	_, err := client.SendSMS(context.Background(), api.SendSMSParams{
		Header:      wenova.DefaultSMSInfoHeader,
		PhoneNumber: "2055123456",
		Message:     "Your appointment is confirmed",
		ScriptID:    123,
		UsePackage:  false,
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

The client sends JSON to `POST /sms/package` on the configured Wenova base URL.

## Troubleshooting

| Problem | Check |
| ------- | ----- |
| `api.ErrMissingCredential` | Pass a token to `wenova.NewWenova` or set `ScriptID` in `api.SendSMSParams` |
| `api.ErrTokenNotFound` | Confirm the token is copied correctly and belongs to the right Wenova script |
| `api.ErrWalletBalanceInsufficient` | Top up wallet balance or use `SendByPackage` if you have package quota |
| `api.ErrPackageQuotaInsufficient` | Use `Send` to charge wallet balance or renew the SMS package |
| `api.ErrLinksNotAllowed` | Remove URLs from the SMS body |
| Request times out | Use a shorter message, check network connectivity, or tune `http.Client.Timeout` |
| Response is not JSON | The API gateway or proxy may have returned an HTML/text error page |

## Production Checklist

- Keep the Wenova token in an environment variable or secret manager.
- Use `context.WithTimeout` or an `http.Client` timeout.
- Log the error code with `api.CodeOf(err)` when a send fails.
- Retry only `api.IsRetryable(err)` errors, and use backoff.
- Do not retry invalid requests, auth failures, or insufficient-fund errors
  without first fixing the cause.
- Avoid sending links unless Wenova has enabled that for your use case.
