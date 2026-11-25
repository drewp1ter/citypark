package iiko

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

// StopListWebhook coalesces notifications for one organization and delivery group.
type StopListWebhook struct {
	token, organizationID, terminalGroupID string
	pending                                chan struct{}
	refresh                                func(context.Context) error
}

func NewStopListWebhook(token, organizationID, terminalGroupID string, refresh func(context.Context) error) *StopListWebhook {
	return &StopListWebhook{
		token: token, organizationID: organizationID, terminalGroupID: terminalGroupID,
		pending: make(chan struct{}, 1), refresh: refresh,
	}
}

func (h *StopListWebhook) Receive(w http.ResponseWriter, r *http.Request) error {
	if h.token == "" {
		return httpx.NotFound()
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(h.token)) != 1 {
		return httpx.Unauthorized()
	}
	var events []struct {
		Type           string          `json:"eventType"`
		OrganizationID string          `json:"organizationId"`
		Info           json.RawMessage `json:"eventInfo"`
	}
	if err := httpx.DecodeJSON(w, r, &events); err != nil {
		return err
	}
	if events == nil {
		return httpx.BadRequest("Некорректный JSON")
	}

	refresh := false
	for _, event := range events {
		if event.Type != "StopListUpdate" || !strings.EqualFold(event.OrganizationID, h.organizationID) {
			continue
		}
		var info struct {
			Groups []struct {
				ID string `json:"id"`
			} `json:"terminalGroupsStopListsUpdates"`
		}
		if err := json.Unmarshal(event.Info, &info); err != nil || info.Groups == nil {
			return httpx.BadRequest("Некорректный JSON")
		}
		for _, group := range info.Groups {
			if group.ID != "" && (h.terminalGroupID == "" || strings.EqualFold(group.ID, h.terminalGroupID)) {
				refresh = true
			}
		}
	}
	if refresh {
		h.enqueue()
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *StopListWebhook) enqueue() {
	select {
	case h.pending <- struct{}{}:
	default:
	}
}

// Run processes notifications independently of their HTTP requests.
func (h *StopListWebhook) Run(ctx context.Context, log *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.pending:
		}
		if ctx.Err() != nil {
			return
		}
		if err := h.refresh(ctx); err != nil && ctx.Err() == nil {
			log.Warn("iiko stop-list refresh failed", "error", err)
			timer := time.NewTimer(30 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				h.enqueue()
			}
		}
	}
}
