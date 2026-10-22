package app_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/app"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/telegram"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const (
	chatCreated   = "111"
	chatConfirmed = "222"
)

type sentMessage struct{ ChatID, Text string }

// fakeSender records every SendMessage call and fails when err is set.
type fakeSender struct {
	mu   sync.Mutex
	sent []sentMessage
	err  error
}

func (f *fakeSender) SendMessage(_ context.Context, chatID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{chatID, text})
	return f.err
}

func (f *fakeSender) calls() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentMessage(nil), f.sent...)
}

func testOrder(paymentType string) order.Order {
	return order.Order{
		ID:            15,
		PaymentType:   paymentType,
		Total:         79000,
		DeliveryPrice: 15000,
		Comment:       "позвонить",
		Customer:      order.Customer{ID: 3, Name: "Иван", Phone: "+79991234567"},
		Address:       &order.Address{ID: 1, City: "Москва", Address: "ул. Ленина, 1"},
		Items:         []order.Item{{ProductID: 7, Name: "Пицца", CategoryName: "Пиццы", Amount: 2, OfferedPrice: 39500}},
	}
}

func testOrderText(paymentType string) string {
	return telegram.FormatOrder(telegram.OrderMessage{
		ID:          15,
		Name:        "Иван",
		Phone:       "+79991234567",
		Address:     "Москва, ул. Ленина, 1",
		PaymentType: paymentType,
		Total:       79000,
		Delivery:    15000,
		Comment:     "позвонить",
		Items:       []telegram.OrderItem{{Category: "Пиццы", Name: "Пицца", Amount: 2}},
	})
}

func newNotifier(sender app.MessageSender, buf *bytes.Buffer) *app.TelegramNotifier {
	log := slog.New(slog.NewTextHandler(buf, nil))
	return app.NewTelegramNotifier(sender, chatCreated, chatConfirmed, log)
}

func TestNotifierCash(t *testing.T) {
	sender := &fakeSender{}
	n := newNotifier(sender, &bytes.Buffer{})

	n.OrderCreated(context.Background(), testOrder(order.PaymentCash))
	n.Wait()

	want := []sentMessage{{chatCreated, testOrderText(order.PaymentCash)}, {chatCreated, "+79991234567"}}
	assertSent(t, sender.calls(), want)
}

func TestNotifierCreatedOnlineIgnored(t *testing.T) {
	sender := &fakeSender{}
	n := newNotifier(sender, &bytes.Buffer{})

	n.OrderCreated(context.Background(), testOrder(order.PaymentOnline))
	n.Wait()

	assertSent(t, sender.calls(), nil)
}

func TestNotifierPaid(t *testing.T) {
	sender := &fakeSender{}
	n := newNotifier(sender, &bytes.Buffer{})

	o := testOrder(order.PaymentOnline)
	n.OrderPaid(context.Background(), o)
	n.Wait()

	want := []sentMessage{
		{chatConfirmed, testOrderText(order.PaymentOnline)},
		{chatCreated, "+79991234567"},
	}
	assertSent(t, sender.calls(), want)
}

func TestNotifierConfirmed(t *testing.T) {
	sender := &fakeSender{}
	n := newNotifier(sender, &bytes.Buffer{})

	n.OrderConfirmed(context.Background(), testOrder(order.PaymentCash))
	n.Wait()

	want := []sentMessage{{chatConfirmed, testOrderText(order.PaymentCash)}, {chatCreated, "+79991234567"}}
	assertSent(t, sender.calls(), want)
}

func TestNotifierSendError(t *testing.T) {
	sender := &fakeSender{err: errors.New("boom")}
	var buf bytes.Buffer
	n := newNotifier(sender, &buf)

	n.OrderCreated(context.Background(), testOrder(order.PaymentCash))
	n.Wait()

	if got := len(sender.calls()); got != 1 {
		t.Errorf("got %d send attempts, want 1", got)
	}
	logged := buf.String()
	for _, want := range []string{"level=ERROR", "telegram send failed", "order_id=15", "boom"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log %q does not contain %q", logged, want)
		}
	}
}

func TestNotifierPhoneSendError(t *testing.T) {
	sender := &failSecond{}
	var buf bytes.Buffer
	n := newNotifier(sender, &buf)

	n.OrderCreated(context.Background(), testOrder(order.PaymentCash))
	n.Wait()

	if got := len(sender.calls()); got != 2 {
		t.Errorf("got %d send attempts, want 2", got)
	}
	if !strings.Contains(buf.String(), "telegram send failed") {
		t.Errorf("log %q has no send error", buf.String())
	}
}

// failSecond succeeds on the first call and fails on every later one.
type failSecond struct{ fakeSender }

func (f *failSecond) SendMessage(ctx context.Context, chatID, text string) error {
	_ = f.fakeSender.SendMessage(ctx, chatID, text)
	if len(f.calls()) > 1 {
		return errors.New("second failed")
	}
	return nil
}

func TestNotifierEmptyChat(t *testing.T) {
	sender := &fakeSender{}
	var buf bytes.Buffer
	n := app.NewTelegramNotifier(sender, "", "", slog.New(slog.NewTextHandler(&buf, nil)))

	n.OrderCreated(context.Background(), testOrder(order.PaymentCash))
	n.OrderPaid(context.Background(), testOrder(order.PaymentOnline))
	n.Wait()

	assertSent(t, sender.calls(), nil)
	if got := strings.Count(buf.String(), "level=WARN"); got != 2 {
		t.Errorf("got %d warnings, want 2: %q", got, buf.String())
	}
}

func TestNotifierPickupHasNoAddress(t *testing.T) {
	sender := &fakeSender{}
	n := newNotifier(sender, &bytes.Buffer{})

	o := testOrder(order.PaymentCash)
	o.Address = nil
	n.OrderCreated(context.Background(), o)
	n.Wait()

	calls := sender.calls()
	if len(calls) == 0 || !strings.Contains(calls[0].Text, "Доставка:       Нет") {
		t.Errorf("pickup order text = %+v, want delivery \"Нет\"", calls)
	}
}

func assertSent(t *testing.T, got, want []sentMessage) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d sends, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("send %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPaymentsAdapter(t *testing.T) {
	p := app.NewMonetaPayments(moneta.New("100", "secret", false))

	form := p.Form(order.PaymentRequest{OrderID: 15, Amount: 79000})
	want := order.PaymentForm{Amount: "790.00", Signature: "8c65fcd4afe3dc4cb915e04aaa273574"}
	if form != want {
		t.Errorf("Form = %+v, want %+v", form, want)
	}

	good := order.PaymentCallback{TransactionID: "15", OperationID: "555", Amount: "790.00", Signature: "998b6268fe3a6c7f4eeb8ccec65a9566"}
	if !p.Verify(good) {
		t.Error("Verify rejected a correct callback")
	}
	bad := good
	bad.Amount = "790.01"
	if p.Verify(bad) {
		t.Error("Verify accepted a callback with a changed amount")
	}
}

// fakeTelegram counts sendMessage requests and records chat ids.
type fakeTelegram struct {
	*httptest.Server
	mu    sync.Mutex
	chats []string
}

func newFakeTelegram(t *testing.T) *fakeTelegram {
	t.Helper()
	f := &fakeTelegram{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			http.NotFound(w, r)
			return
		}
		var body struct {
			ChatID string `json:"chat_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.chats = append(f.chats, body.ChatID)
		f.mu.Unlock()
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeTelegram) sentChats() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.chats...)
}

func newApp(t *testing.T, db *sqlx.DB, tg *fakeTelegram, env map[string]string) *app.App {
	t.Helper()
	merged := map[string]string{
		"IIKO_BASE_URL":                        tg.URL,
		"TELEGRAM_BASE_URL":                    tg.URL,
		"TELEGRAM_TOKEN":                       "tok",
		"TELEGRAM_CHAT_ID_FOR_ORDER_CREATED":   chatCreated,
		"TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED": chatConfirmed,
		"WORKING_HOURS_START":                  "",
		"WORKING_HOURS_END":                    "",
	}
	for k, v := range env {
		merged[k] = v
	}
	cfg, err := config.Load(func(key string) (string, bool) {
		v, ok := merged[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return app.New(cfg, db, log, clock.Fixed(time.Date(2026, 10, 13, 12, 0, 0, 0, cfg.Timezone)))
}

func TestAppSmoke(t *testing.T) {
	db := testutil.NewDB(t)
	a := newApp(t, db, newFakeTelegram(t), nil)

	tests := []struct {
		method, path string
		status       int
		body         string
	}{
		{http.MethodGet, "/api/v1/health", http.StatusOK, ""},
		{http.MethodGet, "/api/v1/categories", http.StatusOK, "[]"},
		{http.MethodGet, "/api/v1/user", http.StatusUnauthorized, ""},
		{http.MethodOptions, "/api/v1/orders", http.StatusNoContent, ""},
		{http.MethodGet, "/nope", http.StatusNotFound, ""},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := testutil.DoJSON(t, a.Handler, tc.method, tc.path, "", nil)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body)
			}
			if tc.body != "" && strings.TrimSpace(rec.Body.String()) != tc.body {
				t.Errorf("body = %s, want %s", rec.Body, tc.body)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
			}
		})
	}

	rec := testutil.DoJSON(t, a.Handler, http.MethodOptions, "/api/v1/orders", "", nil)
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, X-Update-Token, X-Webhook-Secret" {
		t.Errorf("Access-Control-Allow-Headers = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, PATCH, DELETE, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q", got)
	}
}

func TestAppOrderFlow(t *testing.T) {
	db := testutil.NewDB(t)
	tg := newFakeTelegram(t)
	a := newApp(t, db, tg, nil)

	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Пиццы"})
	product := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Пицца", PriceKopecks: 39500})
	testutil.InsertDeliveryPrice(t, db, "Москва", 15000)

	body := map[string]any{
		"phone":       "+79991234567",
		"name":        "Иван",
		"paymentType": "cash",
		"address":     map[string]string{"city": "Москва", "address": "ул. Ленина, 1"},
		"items":       []map[string]any{{"productId": product, "amount": 2}},
	}
	rec := testutil.DoJSON(t, a.Handler, http.MethodPost, "/api/v1/orders", "", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var res order.CreateResult
	testutil.Decode(t, rec, &res)
	if res.OrderID == 0 {
		t.Errorf("result has no order id: %+v", res)
	}

	a.Notifier.Wait()
	got := tg.sentChats()
	if len(got) != 2 || got[0] != chatCreated || got[1] != chatCreated {
		t.Errorf("telegram chats = %v, want [%s %s]", got, chatCreated, chatCreated)
	}
}

func TestAppOnlineFlow(t *testing.T) {
	db := testutil.NewDB(t)
	tg := newFakeTelegram(t)
	a := newApp(t, db, tg, map[string]string{"MONETA_ID": "100", "MONETA_SECRET": "secret"})

	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Пиццы"})
	product := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Пицца", PriceKopecks: 79000})

	body := map[string]any{
		"phone":       "+79991234567",
		"name":        "Иван",
		"paymentType": "online",
		"items":       []map[string]any{{"productId": product, "amount": 1}},
	}
	rec := testutil.DoJSON(t, a.Handler, http.MethodPost, "/api/v1/orders", "", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var res order.CreateResult
	testutil.Decode(t, rec, &res)
	if res.Amount != "790.00" {
		t.Fatalf("amount = %q, want 790.00", res.Amount)
	}

	id := strconv.FormatInt(res.OrderID, 10)
	sum := md5.Sum([]byte("100" + id + "555" + "790.00" + "RUB" + "0" + "secret"))
	callback := func(signature string) *httptest.ResponseRecorder {
		form := url.Values{
			"MNT_TRANSACTION_ID": {id},
			"MNT_OPERATION_ID":   {"555"},
			"MNT_AMOUNT":         {"790.00"},
			"MNT_SIGNATURE":      {signature},
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/moneta", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		a.Handler.ServeHTTP(rec, req)
		return rec
	}

	rec = callback(hex.EncodeToString(sum[:]))
	if rec.Code != http.StatusOK || rec.Body.String() != "SUCCESS" {
		t.Fatalf("callback = %d %q, want 200 SUCCESS", rec.Code, rec.Body)
	}
	var status string
	if err := db.Get(&status, `SELECT status FROM orders WHERE id = $1`, res.OrderID); err != nil {
		t.Fatalf("select status: %v", err)
	}
	if status != order.StatusPaid {
		t.Errorf("status = %q, want %q", status, order.StatusPaid)
	}

	a.Notifier.Wait()
	got := tg.sentChats()
	if len(got) != 2 || got[0] != chatConfirmed || got[1] != chatCreated {
		t.Errorf("telegram chats = %v, want [%s %s]", got, chatConfirmed, chatCreated)
	}

	rec = callback(strings.Repeat("0", 32))
	if rec.Code != http.StatusForbidden || rec.Body.String() != "FAIL" {
		t.Errorf("wrong signature = %d %q, want 403 FAIL", rec.Code, rec.Body)
	}
}

func TestAppUploads(t *testing.T) {
	db := testutil.NewDB(t)
	a := newApp(t, db, newFakeTelegram(t), map[string]string{
		"UPLOAD_DIR":      t.TempDir(),
		"PUBLIC_BASE_URL": "https://api.example.com/",
	})
	adminID := testutil.InsertUser(t, db, "Admin", "+79000000001", auth.RoleAdmin)
	tokens := auth.NewTokens("changeme", time.Hour, clock.Fixed(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)))
	token, err := tokens.Issue(adminID, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	// A PNG signature is enough for http.DetectContentType.
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "a.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(png)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/uploads = %d, body %s", rec.Code, rec.Body)
	}
	var resp struct{ URL string }
	testutil.Decode(t, rec, &resp)
	path, ok := strings.CutPrefix(resp.URL, "https://api.example.com")
	if !ok {
		t.Fatalf("url = %q, want it to start with PUBLIC_BASE_URL", resp.URL)
	}
	if stored, err := a.Uploads.Stored(resp.URL); err != nil || stored != path {
		t.Errorf("Uploads.Stored(%q) = %q, %v; want %q", resp.URL, stored, err, path)
	}

	// The file is served outside /api/v1, for GET and HEAD.
	got := testutil.DoJSON(t, a.Handler, http.MethodGet, path, "", nil)
	if got.Code != http.StatusOK || !bytes.Equal(got.Body.Bytes(), png) {
		t.Errorf("GET %s = %d, body %q", path, got.Code, got.Body)
	}
	head := testutil.DoJSON(t, a.Handler, http.MethodHead, path, "", nil)
	if head.Code != http.StatusOK {
		t.Errorf("HEAD %s = %d, want 200", path, head.Code)
	}
}
