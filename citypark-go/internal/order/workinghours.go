package order

import (
	"context"
	"net/http"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/workinghours"
)

// HoursSource is satisfied by *workinghours.Store; tests use a fake.
type HoursSource interface {
	Get(ctx context.Context) (workinghours.Hours, time.Time, error)
}

// WorkingHours rejects requests outside [start, end), reading the hours on every request.
// A read error is answered with a 500: it never lets the order through.
func WorkingHours(c clock.Clock, hours HoursSource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, _, err := hours.Get(r.Context())
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			if !h.OpenAt(c.Now()) {
				httpx.WriteError(w, r, httpx.NewError(http.StatusServiceUnavailable,
					"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с "+
						workinghours.Display(h.Start)+" до "+workinghours.Display(h.End)))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
