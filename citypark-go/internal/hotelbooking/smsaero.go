package hotelbooking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const smsAeroEndpoint = "https://gate.smsaero.ru/v2/sms/send"

type SMSAero struct {
	endpoint   string
	login, key string
	client     *http.Client
}

func NewSMSAero(login, key string) *SMSAero {
	return &SMSAero{endpoint: smsAeroEndpoint, login: login, key: key, client: &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (s *SMSAero) Send(ctx context.Context, phone, message string) error {
	payload, err := json.Marshal(map[string]string{
		"number":  strings.TrimPrefix(phone, "+"),
		"sign":    "CityPark",
		"text":    message,
		"channel": "DIRECT",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.login, s.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Success bool
		Data    struct{ ID int64 }
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 ||
		json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result) != nil ||
		!result.Success || result.Data.ID <= 0 {
		return errors.New("SMS Aero did not accept the message")
	}
	return nil
}
