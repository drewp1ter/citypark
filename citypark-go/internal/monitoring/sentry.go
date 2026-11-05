// Package monitoring configures opt-in Sentry reporting without a global client.
package monitoring

import (
	"context"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
)

// New constructs an isolated client. An empty DSN disables all reporting.
func New(dsn, environment, release string, rate float64, transport ...sentry.Transport) (*sentry.Client, error) {
	options := sentry.ClientOptions{
		Dsn: dsn, Environment: environment, Release: release,
		EnableTracing: dsn != "", TracesSampleRate: rate, AttachStacktrace: true,
		SendDefaultPII: false, BeforeSend: scrub, BeforeSendTransaction: scrub,
	}
	if len(transport) > 0 {
		options.Transport = transport[0]
	}
	return sentry.NewClient(options)
}

// Middleware installs a fresh hub for each request before the HTTP integration.
// Place it inside the application's existing panic recovery middleware.
func Middleware(client *sentry.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if client.Options().Dsn == "" {
			return next
		}
		routed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				route := "unmatched"
				if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
					route = rc.RoutePattern()
				}
				if tx := sentry.TransactionFromContext(r.Context()); tx != nil {
					tx.Name = r.Method + " " + route
					tx.Source = sentry.SourceRoute
				}
			}()
			next.ServeHTTP(w, r)
		})
		integrated := sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(routed)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hub := sentry.NewHub(client, sentry.NewScope())
			integrated.ServeHTTP(w, r.WithContext(sentry.SetHubOnContext(r.Context(), hub)))
		})
	}
}

// checkReasons are the failure reasons of a MONETA CHECK (order.Check* constants). Only these tag
// values pass scrub; they are copied here because order must not be imported.
var checkReasons = map[string]bool{
	"bad_request": true, "bad_signature": true, "internal_error": true, "invalid_transaction_id": true,
	"order_not_found": true, "order_canceled": true, "order_not_payable": true,
	"retry_not_allowed": true, "amount_mismatch": true,
}

const checkReasonTag = "moneta.check_reason"

// CaptureCheckFailure reports one failed MONETA CHECK, grouped by reason. It carries no order data.
// It does nothing when the context has no hub (Sentry disabled).
func CaptureCheckFailure(ctx context.Context, reason string) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTag(checkReasonTag, reason)
		scope.SetLevel(sentry.LevelWarning)
		hub.CaptureMessage("MONETA check failed: " + reason)
	})
}

// Flush waits for queued telemetry; call after request and background work drains.
func Flush(ctx context.Context, client *sentry.Client) bool {
	return client.FlushWithContext(ctx)
}

// scrub keeps stack locations and trace identifiers, never arbitrary customer data.
// Error text can embed SQL values, addresses or tokens, so values are deliberately
// generic; local application logs retain the original diagnostics.
func scrub(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	route := event.Tags["monitoring.route"]
	checkReason := event.Tags[checkReasonTag]
	event.Request = nil
	event.User = sentry.User{IPAddress: "0.0.0.0"}
	event.Tags = nil
	event.Breadcrumbs = nil
	event.ServerName = ""
	event.Message = ""
	event.Fingerprint = nil
	trace := event.Contexts["trace"]
	event.Contexts = make(map[string]sentry.Context)
	if trace != nil {
		safe := sentry.Context{}
		for _, key := range []string{"trace_id", "span_id", "parent_span_id", "op", "status", "origin"} {
			if value, ok := trace[key]; ok {
				safe[key] = value
			}
		}
		event.Contexts["trace"] = safe
	}
	// Only middleware-normalized route names are permitted for transactions.
	if event.Type != "transaction" {
		event.Transaction = route
		if route != "" {
			event.Fingerprint = []string{"{{ default }}", route}
		}
	}
	for i := range event.Exception {
		ex := &event.Exception[i]
		ex.Value = "Unexpected application error"
		if ex.Mechanism != nil {
			ex.Mechanism.Data = nil
			ex.Mechanism.Description = ""
		}
		if ex.Stacktrace != nil {
			for j := range ex.Stacktrace.Frames {
				ex.Stacktrace.Frames[j].Vars = nil
			}
		}
	}
	for _, span := range event.Spans {
		span.Description = ""
		span.Name = ""
		span.Data = nil
		span.Tags = nil
	}
	if checkReason != "" && checkReasons[checkReason] && len(event.Exception) == 0 && event.Type != "transaction" {
		event.Tags = map[string]string{checkReasonTag: checkReason}
		event.Message = "MONETA check failed: " + checkReason
		event.Fingerprint = []string{"moneta-check", checkReason}
		event.Level = sentry.LevelWarning
		return event
	}
	// String panics use Message rather than Exception; keep a safe issue title.
	if len(event.Exception) == 0 && event.Type != "transaction" {
		event.Message = "Unexpected application error"
	}
	return event
}

// ShutdownTimeout bounds the final telemetry drain independently of HTTP shutdown.
const ShutdownTimeout = 2 * time.Second
