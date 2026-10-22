// Package onec reads the hotel reports from the 1C HTTP services and serves them to the admin panel.
package onec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ErrDisabled is returned by every call of a client created without a base URL.
var ErrDisabled = errors.New("1c is not configured")

// periodLayout is the date format of the StartPeriod and EndPeriod query parameters.
const periodLayout = "20060102150405"

// RoomSale is one row of the "Сдача номеров" report.
type RoomSale struct {
	Organization string  `json:"Organization"`
	Room         string  `json:"Room"`
	Count        float64 `json:"Count"`
	Sum          float64 `json:"Sum"`
}

// DaySale is one row of the "Выручка по дням" report. Date looks like "21.10.2026".
type DaySale struct {
	Date string  `json:"Date"`
	Sum  float64 `json:"Sum"`
}

// Client talks to the 1C HTTP services of the hotel database.
type Client struct {
	base     string
	user     string
	password string
	http     *http.Client
}

// NewClient: httpClient nil → &http.Client{Timeout: 30 * time.Second}. base "" → every call returns ErrDisabled.
func NewClient(base, user, password string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{base: base, user: user, password: password, http: httpClient}
}

// RoomSales returns the revenue by room for the period.
func (c *Client) RoomSales(ctx context.Context, from, to time.Time) ([]RoomSale, error) {
	rows := []RoomSale{}
	if err := c.get(ctx, "room_sales", from, to, &rows); err != nil {
		return nil, err
	}
	if rows == nil { // a null body decodes to nil
		rows = []RoomSale{}
	}
	return rows, nil
}

// DaySales returns the revenue per day for the period. A date can appear in several rows.
func (c *Client) DaySales(ctx context.Context, from, to time.Time) ([]DaySale, error) {
	rows := []DaySale{}
	if err := c.get(ctx, "month_sales", from, to, &rows); err != nil {
		return nil, err
	}
	if rows == nil { // a null body decodes to nil
		rows = []DaySale{}
	}
	return rows, nil
}

// get calls one report service and decodes the JSON array into dst. A null body sets a slice to nil.
func (c *Client) get(ctx context.Context, report string, from, to time.Time, dst any) error {
	if c.base == "" {
		return ErrDisabled
	}
	q := url.Values{}
	q.Set("StartPeriod", from.Format(periodLayout))
	q.Set("EndPeriod", to.Format(periodLayout))
	endpoint := c.base + "/hs/reports/v1/" + report + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("onec: build %s request: %w", report, stripURL(err))
	}
	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("onec: request %s: %w", report, stripURL(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("onec: %s: status %d", report, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("onec: decode %s: %w", report, err)
	}
	return nil
}

// stripURL drops the request URL from a *url.Error, so the address of the 1C server stays out of the error text.
func stripURL(err error) error {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		return urlErr.Err
	}
	return err
}
