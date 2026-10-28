package live

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
)

const (
	writeTimeout = 5 * time.Second
	pingEvery    = 30 * time.Second
	pingTimeout  = 10 * time.Second
)

type Handler struct {
	hub    *Hub
	tokens *auth.Tokens
	log    *slog.Logger
}

func NewHandler(hub *Hub, tokens *auth.Tokens, log *slog.Logger) *Handler {
	return &Handler{hub: hub, tokens: tokens, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/orders/live", h.serve)
}

// token reads the JWT from "Sec-WebSocket-Protocol: bearer, <jwt>". A browser cannot set Authorization on a WebSocket.
func token(r *http.Request) (string, bool) {
	var parts []string
	for _, v := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, p := range strings.Split(v, ",") {
			parts = append(parts, strings.TrimSpace(p))
		}
	}
	if len(parts) != 2 || parts[0] != "bearer" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// serve is a plain handler, not httpx.Handle: after the upgrade there is no HTTP response to write.
func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	tok, ok := token(r)
	if !ok {
		httpx.WriteError(w, r, httpx.Unauthorized())
		return
	}
	claims, err := h.tokens.Parse(tok)
	if err != nil {
		httpx.WriteError(w, r, httpx.Unauthorized())
		return
	}
	if !auth.Allows(claims.Role, auth.RoleAdmin) {
		httpx.WriteError(w, r, httpx.Forbidden())
		return
	}

	// The token is a header, not a cookie, so another site cannot ride an admin's session: no origin check.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{"bearer"},
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.log.Warn("live websocket accept failed", "error", err)
		return
	}

	events, unsubscribe := h.hub.Subscribe()
	defer unsubscribe()
	// The client sends nothing; CloseRead handles its close frames and cancels ctx when it goes away.
	ctx := conn.CloseRead(r.Context())

	if err := write(ctx, conn, Event{Type: "hello"}); err != nil {
		_ = conn.CloseNow()
		return
	}

	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	for {
		select {
		case e, open := <-events:
			if !open {
				_ = conn.Close(websocket.StatusTryAgainLater, "reconnect")
				return
			}
			if err := write(ctx, conn, e); err != nil {
				_ = conn.CloseNow()
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				_ = conn.CloseNow()
				return
			}
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		}
	}
}

func write(ctx context.Context, conn *websocket.Conn, e Event) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return wsjson.Write(ctx, conn, e)
}
