# laogo

A collection of small, dependency-free Go libraries for Lao (Laos) developers —
integrations with local services and providers, packaged for easy reuse.

```bash
go get github.com/barluscuda/laogo
```

```go
import "github.com/barluscuda/laogo"

laogo.Version() // "1.0-beta"
```

Requires Go 1.26+.

## Packages

| Package | Import path | Description |
| ------- | ----------- | ----------- |
| SMS Gateway — Wenova | `github.com/barluscuda/laogo/sms-gateway/wenova` | Client for sending SMS/OTP messages through the [Wenova](https://microservices.wenova.fun/) API. See [sms-gateway/wenova/README.md](./sms-gateway/wenova/README.md). |

## Testing

```bash
go test ./...
```

## License

See repository for details.
