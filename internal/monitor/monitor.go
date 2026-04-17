package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"i9pay/internal/config"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// sentryEnabled tracks whether Sentry was successfully initialized.
var sentryEnabled bool

// Init initializes the monitoring backend. Safe to call even when DSN is empty
// (e.g. local development) — in that case it is a no-op.
func Init(cfg *config.Config) {
	if cfg.SentryDSN == "" {
		return
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.GoEnv,
		AttachStacktrace: true,
		TracesSampleRate: 0.2,
	}); err != nil {
		fmt.Printf("monitor: initialization failed: %v\n", err)
		return
	}
	sentryEnabled = true
}

// Flush waits for buffered events to be sent to Sentry. Should be called
// before the process exits to avoid losing in-flight error reports.
func Flush() {
	sentry.Flush(2 * time.Second)
}

// Middleware returns a Gin middleware that instruments each request for error
// tracking. Panics are captured and re-raised so the recovery middleware can
// still produce a proper HTTP response.
// When Sentry is not initialized (e.g. no DSN in dev), a no-op is returned
// to avoid nil pointer dereferences inside sentrygin.
func Middleware() gin.HandlerFunc {
	if !sentryEnabled {
		return func(c *gin.Context) { c.Next() }
	}
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

// LogHandler is an slog.Handler that forwards log records to Sentry in
// addition to the wrapped inner handler.
// - Error level  → captured as a Sentry event via the request-scoped hub (falls back to global).
// - Warn / Info  → added as a breadcrumb on the request-scoped hub (falls back to global).
//
// Using the request-scoped hub keeps breadcrumbs and events correlated to the
// same transaction that sentrygin creates, avoiding duplicate events.
type LogHandler struct {
	inner slog.Handler
}

// NewLogHandler wraps the given slog.Handler and adds Sentry integration.
func NewLogHandler(inner slog.Handler) slog.Handler {
	return &LogHandler{inner: inner}
}

// Enabled implements slog.Handler.
func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle implements slog.Handler. It forwards the record to the inner handler
// and additionally sends it to Sentry: errors as events, all others as breadcrumbs.
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	extra := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		extra[a.Key] = a.Value.Any()
		return true
	})

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	level := sentry.LevelInfo
	if r.Level >= slog.LevelWarn {
		level = sentry.LevelWarning
	}

	// Always add a breadcrumb so there is an audit trail for every log line.
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Level:   level,
		Message: r.Message,
		Data:    extra,
	}, nil)

	// Errors are additionally sent as standalone events so they appear in
	// Sentry Issues. CaptureException (called by the Errors middleware) already
	// does this for HTTP-level errors, but logger.Error() calls that happen
	// outside a request context (e.g. startup failures) still need coverage.
	if r.Level >= slog.LevelError {
		event := sentry.NewEvent()
		event.Level = sentry.LevelError
		event.Message = r.Message
		event.Extra = extra
		hub.CaptureEvent(event)
	}

	return h.inner.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{inner: h.inner.WithGroup(name)}
}
