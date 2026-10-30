package customer

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
)

type limitEntry struct {
	start time.Time
	count int
}
type Limits struct {
	mu      sync.Mutex
	clock   clock.Clock
	entries map[string]limitEntry
}

func NewLimits(c clock.Clock) *Limits { return &Limits{clock: c, entries: make(map[string]limitEntry)} }
func emailKey(email string) string {
	h := sha256.Sum256([]byte(email))
	return hex.EncodeToString(h[:])
}
func directIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func (l *Limits) allow(key string, limit int, window time.Duration) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	for k, v := range l.entries {
		if !now.Before(v.start.Add(time.Hour)) {
			delete(l.entries, k)
		}
	}
	v := l.entries[key]
	if v.start.IsZero() || !now.Before(v.start.Add(window)) {
		if _, exists := l.entries[key]; !exists && len(l.entries) >= 10000 {
			return false, 60
		}
		v = limitEntry{start: now}
	}
	if v.count >= limit {
		return false, max(1, int(v.start.Add(window).Sub(now).Seconds()+0.999))
	}
	v.count++
	l.entries[key] = v
	return true, 0
}
func (l *Limits) SignIn(email string, r *http.Request) (bool, int) {
	a, x := l.allow("sign-email:"+emailKey(email), 10, 15*time.Minute)
	b, y := l.allow("sign-ip:"+directIP(r), 30, 15*time.Minute)
	return a && b, max(x, y)
}
func (l *Limits) Send(email string, r *http.Request) bool {
	a, _ := l.allow("send-email:"+emailKey(email), 3, time.Hour)
	b, _ := l.allow("send-ip:"+directIP(r), 10, time.Hour)
	return a && b
}
func (l *Limits) Confirm(r *http.Request) (bool, int) {
	return l.allow("confirm-ip:"+directIP(r), 20, 15*time.Minute)
}
func (l *Limits) Reset(r *http.Request) (bool, int) {
	return l.allow("reset-ip:"+directIP(r), 20, 15*time.Minute)
}
func (l *Limits) GoogleStart(r *http.Request) (bool, int) {
	return l.allow("google-start-ip:"+directIP(r), 20, 15*time.Minute)
}
func (l *Limits) GoogleComplete(r *http.Request) (bool, int) {
	return l.allow("google-complete-ip:"+directIP(r), 20, 15*time.Minute)
}
func (l *Limits) Telegram(r *http.Request) (bool, int) {
	return l.allow("telegram-ip:"+directIP(r), 30, 15*time.Minute)
}
func retryAfter(w http.ResponseWriter, seconds int) {
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}

func (l *Limits) PairCreate(userID int64) (bool, int) {
	return l.allow("pair-create:"+strconv.FormatInt(userID, 10), 5, time.Hour)
}
func (l *Limits) PairRedeem(userID int64, r *http.Request) (bool, int) {
	a, x := l.allow("pair-redeem-ip:"+directIP(r), 10, 15*time.Minute)
	b, y := l.allow("pair-redeem-user:"+strconv.FormatInt(userID, 10), 5, 15*time.Minute)
	return a && b, max(x, y)
}
func (l *Limits) PairConfirm(userID int64) (bool, int) {
	return l.allow("pair-confirm:"+strconv.FormatInt(userID, 10), 20, 15*time.Minute)
}
