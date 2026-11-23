package google

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/jmoiron/sqlx"
)

const yandexProfileURL = "https://login.yandex.ru/info?format=json"

func NewYandexService(db *sqlx.DB, c clock.Clock, clientID, secret, callback string) *Service {
	s := NewService(db, c, clientID, callback, "https://oauth.yandex.ru/authorize")
	s.method, s.label, s.clientSecret = "yandex", "Яндекс ID", secret
	s.httpClient = &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return s
}

func (s *Service) exchangeYandex(ctx context.Context, code, verifier string) (profile, error) {
	form := url.Values{"code": {code}, "client_id": {s.clientID}, "client_secret": {s.clientSecret}, "code_verifier": {verifier}, "grant_type": {"authorization_code"}}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	_, err := s.getJSON(ctx, http.MethodPost, "https://oauth.yandex.ru/token", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &token)
	if errors.Is(err, errProviderRejected) {
		return profile{}, s.invalidFlow()
	}
	if err != nil {
		return profile{}, s.unavailable()
	}
	if token.AccessToken == "" {
		return profile{}, s.invalidFlow()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, yandexProfileURL, nil)
	if err != nil {
		return profile{}, s.unavailable()
	}
	req.Header.Set("Authorization", "OAuth "+token.AccessToken)
	res, err := s.httpClient.Do(req)
	if err != nil {
		return profile{}, s.unavailable()
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return profile{}, s.invalidFlow()
	}
	var account struct {
		ID       string `json:"id"`
		ClientID string `json:"client_id"`
		Name     string `json:"real_name"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&account) != nil {
		return profile{}, s.unavailable()
	}
	if account.ClientID != s.clientID || account.ID == "" {
		return profile{}, s.invalidFlow()
	}
	return profile{Subject: account.ID, Name: account.Name}, nil
}
