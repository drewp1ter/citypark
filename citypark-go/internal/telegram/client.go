// Package telegram sends messages through the Telegram Bot API and formats the order text.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client talks to the Telegram Bot API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient: baseURL e.g. "https://api.telegram.org". A nil httpClient means a client with a 15 second timeout.
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: baseURL, token: token, http: httpClient}
}

// SendMessage posts a Markdown message to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"parse_mode": "Markdown",
		"text":       text,
	})
	if err != nil {
		return fmt.Errorf("telegram: encode request: %w", err)
	}

	endpoint := c.baseURL + "/bot" + c.token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", stripURL(err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send message: %w", stripURL(err))
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("telegram: server error, status %d", resp.StatusCode)
	case resp.StatusCode >= http.StatusBadRequest:
		var out struct {
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return fmt.Errorf("telegram: status %d", resp.StatusCode)
		}
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

// stripURL drops the request URL from a *url.Error. The URL contains the bot token.
func stripURL(err error) error {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		return urlErr.Err
	}
	return err
}
