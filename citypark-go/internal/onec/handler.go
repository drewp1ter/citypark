package onec

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
)

const (
	maxPeriodDays = 366
	paramDate     = "2006-01-02"
	onecDate      = "02.01.2006"
)

// Handler serves the hotel report endpoints.
type Handler struct {
	client *Client
	tokens *auth.Tokens
	clock  clock.Clock
}

func NewHandler(client *Client, tokens *auth.Tokens, c clock.Clock) *Handler {
	return &Handler{client: client, tokens: tokens, clock: c}
}

// Routes registers the admin report routes.
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Get("/reports/hotel/sales", httpx.Handle(h.sales))
		r.Get("/reports/hotel/daily", httpx.Handle(h.daily))
	})
}

type saleRow struct {
	Organization string        `json:"organization"`
	Room         string        `json:"room"`
	Count        float64       `json:"count"`
	Sum          money.Kopecks `json:"sum"`
}

type salesTotal struct {
	Count float64       `json:"count"`
	Sum   money.Kopecks `json:"sum"`
}

type salesResponse struct {
	Rows  []saleRow  `json:"rows"`
	Total salesTotal `json:"total"`
}

type dayRow struct {
	Date string        `json:"date"`
	Sum  money.Kopecks `json:"sum"`
}

type dailyResponse struct {
	Days  []dayRow      `json:"days"`
	Total money.Kopecks `json:"total"`
}

func (h *Handler) sales(w http.ResponseWriter, r *http.Request) error {
	from, to, err := h.period(r)
	if err != nil {
		return err
	}
	sales, err := h.client.RoomSales(r.Context(), from, to)
	if err != nil {
		return upstreamError(err)
	}

	resp := salesResponse{Rows: make([]saleRow, 0, len(sales))}
	for _, s := range sales {
		row := saleRow{Organization: s.Organization, Room: s.Room, Count: s.Count, Sum: money.FromRubles(s.Sum)}
		resp.Rows = append(resp.Rows, row)
		resp.Total.Count += row.Count
		resp.Total.Sum += row.Sum
	}
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		a, b := resp.Rows[i], resp.Rows[j]
		if a.Organization != b.Organization {
			return a.Organization < b.Organization
		}
		return a.Room < b.Room
	})
	httpx.JSON(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) daily(w http.ResponseWriter, r *http.Request) error {
	from, to, err := h.period(r)
	if err != nil {
		return err
	}
	sales, err := h.client.DaySales(r.Context(), from, to)
	if err != nil {
		return upstreamError(err)
	}

	// Sum the 1C rows per calendar day. Rows outside the period are dropped below.
	sums := make(map[string]float64)
	for _, s := range sales {
		d, err := time.Parse(onecDate, s.Date)
		if err != nil {
			slog.Default().Error("1c report returned an invalid date", "date", s.Date, "error", err)
			return badGateway()
		}
		sums[d.Format(paramDate)] += s.Sum
	}

	resp := dailyResponse{Days: []dayRow{}}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(paramDate)
		row := dayRow{Date: key, Sum: money.FromRubles(sums[key])}
		resp.Days = append(resp.Days, row)
		resp.Total += row.Sum
	}
	httpx.JSON(w, http.StatusOK, resp)
	return nil
}

// period reads and checks from and to. It returns the start of the first day and the end of the last day
// in the clock's location.
func (h *Handler) period(r *http.Request) (from, to time.Time, err error) {
	loc := h.clock.Now().Location()
	fields := map[string]string{}

	q := r.URL.Query()
	from, errFrom := time.ParseInLocation(paramDate, q.Get("from"), loc)
	to, errTo := time.ParseInLocation(paramDate, q.Get("to"), loc)
	if errFrom != nil {
		fields["from"] = "дата в формате ГГГГ-ММ-ДД"
	}
	if errTo != nil {
		fields["to"] = "дата в формате ГГГГ-ММ-ДД"
	}
	if len(fields) == 0 {
		switch {
		case to.Before(from):
			fields["to"] = "не раньше даты from"
		case to.After(from.AddDate(0, 0, maxPeriodDays-1)):
			fields["to"] = "период не больше 366 дней"
		}
	}
	if len(fields) > 0 {
		e := httpx.BadRequest("Некорректные параметры отчёта")
		e.Fields = fields
		return time.Time{}, time.Time{}, e
	}
	return from, to.Add(24*time.Hour - time.Second), nil
}

// upstreamError maps a client error to the HTTP error.
func upstreamError(err error) error {
	if errors.Is(err, ErrDisabled) {
		return httpx.NotFound()
	}
	slog.Default().Error("1c request failed", "error", err)
	return badGateway()
}

func badGateway() *httpx.Error {
	return httpx.NewError(http.StatusBadGateway, "Сервис отчётов недоступен")
}
