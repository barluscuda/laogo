# Wenova SMS Gateway

A small, dependency-free Go client for sending SMS/OTP messages through the
[Wenova](https://wenova.fun) API.

- `wenova` — high-level client with sensible defaults.
- `wenova/api` — low-level HTTP client, input validation, and typed errors.

For internal workflow and diagrams, see [ARCHITECTURE.md](./ARCHITECTURE.md).

## Install

```bash
go get github.com/barluscuda/laogo/sms-gateway/wenova
```

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/barluscuda/laogo/sms-gateway/wenova"
	"github.com/barluscuda/laogo/sms-gateway/wenova/api"
)

func main() {
	wv := wenova.NewWenova("YOUR_SCRIPT_TOKEN")

	resp, err := wv.Send(context.Background(), wenova.DefaultSMSOTPHeader, "2055123456", "Your code is 123456")
	if err != nil {
		switch {
		case errors.Is(err, api.ErrRateLimitExceeded):
			log.Println("rate limited, retry later")
		case errors.Is(err, api.ErrWalletBalanceInsufficient):
			log.Println("top up wallet")
		default:
			log.Fatalf("send failed: %v", err)
		}
		return
	}
	fmt.Printf("sent: %+v\n", resp)
}
```

## Sending

The header is yours to choose. Two convenient defaults are exported:

| Constant                       | Value      |
| ------------------------------ | ---------- |
| `wenova.DefaultSMSInfoHeader`  | `WNV-info` |
| `wenova.DefaultSMSOTPHeader`   | `WNV-OTP`  |

Two send methods, differing only in how the cost is charged:

```go
// Charge the wallet balance.
wv.Send(ctx, header, phone, message)

// Deduct from the SMS package.
wv.SendByPackage(ctx, header, phone, message)
```

Both return `(*api.Response, error)`.

## Configuration

`NewWenova` takes functional options:

```go
wv := wenova.NewWenova(token,
	wenova.WithBaseURL("https://staging.example.com"), // override the API base URL
	wenova.WithClient(api.NewClient("")),              // supply a custom api.Client
)
```

To tune timeouts or transport, build the `api.Client` yourself:

```go
hc := &http.Client{Timeout: 10 * time.Second}
wv := wenova.NewWenova(token, wenova.WithClient(&api.Client{HTTPClient: hc}))
```

## Error handling

Failures from the API come back as `*api.APIError`, carrying the Wenova business
code (the 5-digit `30xxx` id from the JSON body — not the HTTP status).

### Match a specific failure with `errors.Is`

```go
if errors.Is(err, api.ErrPackageQuotaInsufficient) {
	// SMS package is empty
}
```

Sentinels (all `*api.APIError`):

| Sentinel                          | Code    | Meaning                                  |
| --------------------------------- | ------- | ---------------------------------------- |
| `api.ErrTokenOrScriptIDRequired`  | `30101` | Either token or scriptId is required     |
| `api.ErrLinksNotAllowed`          | `30102` | Links are not allowed in SMS messages    |
| `api.ErrPackageQuotaInsufficient` | `30105` | SMS package quota is insufficient        |
| `api.ErrWalletBalanceInsufficient`| `30106` | Wallet balance is insufficient           |
| `api.ErrWalletNotFound`           | `30302` | Wallet not found for user                |
| `api.ErrPackageNotFound`          | `30303` | SMS package not found                    |
| `api.ErrTokenNotFound`            | `30307` | SMS API token not found                  |
| `api.ErrScriptIDNotFound`         | `30308` | SMS scriptId not found                   |
| `api.ErrRateLimitExceeded`        | `30501` | SMS rate limit exceeded                  |
| `api.ErrGatewayError`             | `30901` | SMS gateway returned an error            |

### Handle a class of failures by category

```go
switch api.CategoryOf(err) {
case api.CategoryRateLimited, api.CategoryGateway:
	// transient — retry with backoff
case api.CategoryInsufficientFunds:
	// top up wallet or package
case api.CategoryAuth:
	// bad token/scriptId
case api.CategoryInvalidRequest:
	// fix the request
}
```

Helpers:

| Function                          | Returns                                     |
| --------------------------------- | ------------------------------------------- |
| `api.CategoryOf(err)`             | the `api.Category` bucket                    |
| `api.IsRetryable(err)`            | `true` for rate-limit / gateway errors       |
| `api.CodeOf(err)`                 | the raw Wenova code, or `0`                   |
| `api.AsAPIError(err)`             | `(*api.APIError, bool)` (unwraps)             |

### Client-side validation errors

Returned before any request is made; also matchable with `errors.Is`:

`api.ErrMissingHeader`, `api.ErrMissingPhone`, `api.ErrMissingMessage`,
`api.ErrMissingCredential`.

## Testing

```bash
go test ./sms-gateway/...
```
