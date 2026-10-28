package iiko

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ErrUnauthorized is returned when iiko answers 401.
var ErrUnauthorized = errors.New("iiko: unauthorized")

// StatusError: iiko answered with a status other than 200 and 401.
type StatusError struct {
	Endpoint    string
	Status      int
	Description string // errorDescription from the body, "" when the body has none
}

func (e *StatusError) Error() string {
	msg := fmt.Sprintf("iiko: %s: unexpected status %d", e.Endpoint, e.Status)
	if e.Description != "" {
		msg += ": " + e.Description
	}
	return msg
}

// Credentials authenticate the application in iiko.
type Credentials struct{ APIKey, AppID, ClientSecret string }

// Client talks to the iiko Cloud API.
type Client struct {
	baseURL        string
	creds          Credentials
	organizationID string
	externalMenuID string
	http           *http.Client
	pollEvery      time.Duration // pause before each commands/status check
	pollTimes      int           // how many checks CreateDelivery makes at most

	mu    sync.Mutex
	token string
}

// NewClient: baseURL has no trailing slash, e.g. "https://api-ru.iiko.services/api".
// A nil httpClient means a client with a 60 second timeout.
func NewClient(baseURL string, creds Credentials, organizationID, externalMenuID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		baseURL:        baseURL,
		creds:          creds,
		organizationID: organizationID,
		externalMenuID: externalMenuID,
		http:           httpClient,
		pollEvery:      2 * time.Second,
		pollTimes:      15,
	}
}

// AccessToken requests a new token.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	body := map[string]string{
		"apiKey":       c.creds.APIKey,
		"appId":        c.creds.AppID,
		"clientSecret": c.creds.ClientSecret,
	}
	if err := c.post(ctx, "v2/access_token", "", body, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", errors.New("iiko: v2/access_token: empty token")
	}
	return out.Token, nil
}

// Menu downloads the external menu using the given token.
func (c *Client) Menu(ctx context.Context, token string) (*Menu, error) {
	body := map[string]string{"externalMenuId": c.externalMenuID, "organizationId": c.organizationID}
	var out Menu
	if err := c.post(ctx, "menu/v3/by_id", token, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Fetch returns the external menu, managing the token itself (withToken).
func (c *Client) Fetch(ctx context.Context) (*Menu, error) {
	var m *Menu
	err := c.withToken(ctx, func(token string) error {
		var err error
		m, err = c.Menu(ctx, token)
		return err
	})
	return m, err
}

// withToken runs call with the cached token, getting one first if needed.
// If call returns ErrUnauthorized, it gets a new token and runs call once more.
// It holds c.mu for the whole call, so one iiko call runs at a time.
func (c *Client) withToken(ctx context.Context, call func(token string) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token == "" {
		token, err := c.AccessToken(ctx)
		if err != nil {
			return err
		}
		c.token = token
	}

	err := call(c.token)
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		return err
	}

	// The cached token expired: get a new one and try once more.
	c.token = ""
	token, err := c.AccessToken(ctx)
	if err != nil {
		return err
	}
	c.token = token
	return call(c.token)
}

// post sends a JSON request to {baseURL}/{endpoint} and decodes a 200 answer into out.
func (c *Client) post(ctx context.Context, endpoint, token string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("iiko: %s: encode request: %w", endpoint, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("iiko: %s: build request: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("iiko: %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		var failure struct {
			ErrorDescription string `json:"errorDescription"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&failure) // the body is optional
		return &StatusError{Endpoint: endpoint, Status: resp.StatusCode, Description: failure.ErrorDescription}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("iiko: %s: decode response: %w", endpoint, err)
	}
	return nil
}
