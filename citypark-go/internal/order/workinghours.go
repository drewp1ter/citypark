package order

import (
	"net/http"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
)

// WorkingHours rejects requests outside [start, end). start and end look like "9:00".
// If either is empty or cannot be parsed, every request passes.
func WorkingHours(c clock.Clock, start, end string) func(http.Handler) http.Handler {
	s, okStart := salewindow.ParseClock(start)
	e, okEnd := salewindow.ParseClock(end)
	if !okStart || !okEnd {
		return func(next http.Handler) http.Handler { return next }
	}

	window := salewindow.Window{Start: &s, End: &e}
	closed := httpx.NewError(http.StatusServiceUnavailable,
		"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с "+start+" до "+end)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !window.OpenAtTime(c.Now()) {
				httpx.WriteError(w, r, closed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
