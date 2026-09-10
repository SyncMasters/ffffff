package network

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/johan-larp/agentsearch/internal/security"
)

// NewRetryableClient оборачивает http.Client в retryablehttp.Client.
// Автоматически повторяет запросы при:
//   - 429 Too Many Requests
//   - 5xx Server Errors
//   - network timeouts
//   - connection errors
//
// Использует exponential backoff с jitter для снижения нагрузки на сервер.
func NewRetryableClient(base *http.Client, maxRetries int) *http.Client {
	if maxRetries <= 0 {
		maxRetries = 2
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = base
	retryClient.RetryMax = maxRetries
	retryClient.RetryWaitMin = 500 * time.Millisecond
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.Logger = nil // отключаем внутренний логгер, используем slog

	// Кастомный checker: логируем причины retry
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		shouldRetry, checkErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		if shouldRetry && resp != nil {
			rawURL := ""
			if resp.Request != nil && resp.Request.URL != nil {
				rawURL = resp.Request.URL.String()
			}
			slog.Warn("retrying request",
				"status", resp.StatusCode,
				"url", security.Redact(rawURL),
				"error", security.Redact(fmt.Sprint(err)),
			)
		}
		return shouldRetry, checkErr
	}

	// Оборачиваем стандартный RoundTripper для совместимости
	return retryClient.StandardClient()
}
