package customer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

var errTelegramProof = errors.New("invalid telegram proof")

const telegramFailure = "Не удалось подтвердить вход через Telegram"

type TelegramLaunch struct {
	ID        int64
	Name      string
	AuthDate  time.Time
	ProofHash []byte
}

// ValidateTelegramInitData checks Telegram's bot-token HMAC over decoded fields.
// The signature field participates in this HMAC; only hash is excluded.
func ValidateTelegramInitData(raw, botToken string, now time.Time) (TelegramLaunch, error) {
	if raw == "" || len(raw) > 16<<10 || botToken == "" {
		return TelegramLaunch{}, errTelegramProof
	}
	fields := make(map[string]string)
	for _, pair := range strings.Split(raw, "&") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return TelegramLaunch{}, errTelegramProof
		}
		k, err := url.QueryUnescape(key)
		if err != nil || k == "" || !utf8.ValidString(k) {
			return TelegramLaunch{}, errTelegramProof
		}
		v, err := url.QueryUnescape(value)
		if err != nil || !utf8.ValidString(v) {
			return TelegramLaunch{}, errTelegramProof
		}
		if _, exists := fields[k]; exists {
			return TelegramLaunch{}, errTelegramProof
		}
		fields[k] = v
	}
	hash, err := hex.DecodeString(fields["hash"])
	if err != nil || len(hash) != sha256.Size || fields["auth_date"] == "" || fields["user"] == "" {
		return TelegramLaunch{}, errTelegramProof
	}
	keys := make([]string, 0, len(fields)-1)
	for k := range fields {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	check := strings.Join(lines, "\n")
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	proofMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = proofMAC.Write([]byte(check))
	if !hmac.Equal(hash, proofMAC.Sum(nil)) {
		return TelegramLaunch{}, errTelegramProof
	}
	seconds, err := strconv.ParseInt(fields["auth_date"], 10, 64)
	if err != nil || seconds <= 0 {
		return TelegramLaunch{}, errTelegramProof
	}
	authDate := time.Unix(seconds, 0)
	if now.Sub(authDate) > 5*time.Minute || authDate.Sub(now) > 30*time.Second {
		return TelegramLaunch{}, errTelegramProof
	}
	var person struct {
		ID        json.RawMessage `json:"id"`
		FirstName string          `json:"first_name"`
		LastName  string          `json:"last_name"`
	}
	decoder := json.NewDecoder(strings.NewReader(fields["user"]))
	if err := decoder.Decode(&person); err != nil {
		return TelegramLaunch{}, errTelegramProof
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return TelegramLaunch{}, errTelegramProof
	}
	if len(person.ID) == 0 || person.ID[0] < '0' || person.ID[0] > '9' {
		return TelegramLaunch{}, errTelegramProof
	}
	id, err := strconv.ParseInt(string(person.ID), 10, 64)
	if err != nil || id <= 0 || id > (1<<52)-1 {
		return TelegramLaunch{}, errTelegramProof
	}
	name := strings.TrimSpace(person.FirstName + " " + person.LastName)
	if name == "" {
		name = "Пользователь"
	}
	for utf8.RuneCountInString(name) > 80 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	proof := sha256.Sum256([]byte(check))
	return TelegramLaunch{ID: id, Name: name, AuthDate: authDate, ProofHash: proof[:]}, nil
}

type TelegramHandler struct {
	db       *sqlx.DB
	sessions *Service
	limits   *Limits
	token    string
	enabled  bool
}

func NewTelegramHandler(db *sqlx.DB, sessions *Service, limits *Limits, token string, enabled bool) *TelegramHandler {
	return &TelegramHandler{db: db, sessions: sessions, limits: limits, token: token, enabled: enabled}
}
func (h *TelegramHandler) Routes(r chi.Router) {
	r.Post("/auth/customer/telegram", httpx.Handle(h.signIn))
}
func (h *TelegramHandler) signIn(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "private, no-store")
	if !h.enabled || h.token == "" {
		return httpx.NewError(503, "Вход через Telegram недоступен")
	}
	var body struct {
		InitData string `json:"initData"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if len(body.InitData) > 16<<10 {
		return httpx.BadRequest("Некорректный JSON")
	}
	if ok, retry := h.limits.Telegram(r); !ok {
		retryAfter(w, retry)
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	launch, err := ValidateTelegramInitData(body.InitData, h.token, h.sessions.clock.Now())
	if err != nil {
		return httpx.NewError(401, telegramFailure)
	}
	var result struct {
		User      user.Response `json:"user"`
		Token     string        `json:"token"`
		ExpiresAt time.Time     `json:"expiresAt"`
	}
	err = store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		ctx := r.Context()
		if _, e := tx.ExecContext(ctx, `DELETE FROM customer_telegram_proofs WHERE expires_at<=$1`, h.sessions.clock.Now()); e != nil {
			return e
		}
		// Keep each proof until its acceptance window and future-skew window end.
		if _, e := tx.ExecContext(ctx, `INSERT INTO customer_telegram_proofs(proof_hash,expires_at) VALUES($1,$2)`, launch.ProofHash, launch.AuthDate.Add(5*time.Minute+time.Second)); e != nil {
			return e
		}
		subject := strconv.FormatInt(launch.ID, 10)
		if _, e := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "telegram:"+subject); e != nil {
			return e
		}
		st := NewStore(tx)
		id, e := st.IdentityOwner(ctx, "telegram", subject)
		if errors.Is(e, store.ErrNotFound) {
			u, createErr := user.NewStore(tx).Create(ctx, user.CreateParams{Name: launch.Name, Role: auth.RoleUser})
			if createErr != nil {
				return createErr
			}
			id = u.ID
			e = st.AddIdentity(ctx, Identity{UserID: id, Provider: "telegram", Subject: subject})
		}
		if e != nil {
			return e
		}
		u, e := user.NewStore(tx).ByID(ctx, id)
		if e != nil {
			return e
		}
		result.User = u.Response()
		result.Token, result.ExpiresAt, e = h.sessions.createSession(ctx, st, id, "telegram", "mini")
		return e
	})
	if store.IsUniqueViolation(err) {
		return httpx.NewError(401, telegramFailure)
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, result)
	return nil
}
