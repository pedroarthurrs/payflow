package monitor

import (
	"fmt"

	"i9pay/internal/config"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// Init initializes the monitoring backend. Safe to call even when DSN is empty
// (e.g. local development) — in that case it is a no-op.
func Init(cfg *config.Config) {
	if cfg.SentryDSN == "" {
		return
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryDSN,
		Environment: cfg.GoEnv,
	}); err != nil {
		fmt.Printf("monitor: initialization failed: %v\n", err)
	}
}

// Middleware returns a Gin middleware that instruments each request for error
// tracking. Panics are captured and re-raised so the recovery middleware can
// still produce a proper HTTP response.
func Middleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{Repanic: true})
}

// CaptureException reports an error to the monitoring backend using the
// request-scoped hub. If no hub is available (e.g. in tests or when the DSN
// is empty) the call is silently ignored.
func CaptureException(c *gin.Context, err error) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
	}
}
