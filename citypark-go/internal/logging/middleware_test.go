package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// logLine decodes the single JSON log line written to buf.
func logLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("log output is not one JSON line: %v\n%s", err, buf.String())
	}
	return m
}

// serve runs h wrapped in Requests against req and returns the log line.
func serve(t *testing.T, h http.HandlerFunc, req *http.Request) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	log := New(&buf, "production", false)
	Requests(log)(h).ServeHTTP(httptest.NewRecorder(), req)
	return logLine(t, &buf)
}

func TestRequestsLogsStatus(t *testing.T) {
	line := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}, httptest.NewRequest(http.MethodPost, "/x", nil))

	if line["status"] != float64(201) {
		t.Errorf("status = %v, want 201", line["status"])
	}
	if line["method"] != "POST" {
		t.Errorf("method = %v, want POST", line["method"])
	}
	if line["path"] != "/x" {
		t.Errorf("path = %v, want /x", line["path"])
	}
	if line["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", line["level"])
	}
	if line["msg"] != "request" {
		t.Errorf("msg = %v, want request", line["msg"])
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Error("duration_ms is missing")
	}
}

func TestRequestsDefaultStatus(t *testing.T) {
	line := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}, httptest.NewRequest(http.MethodGet, "/x", nil))

	if line["status"] != float64(200) {
		t.Errorf("status = %v, want 200", line["status"])
	}
}

func TestRequestsNoWrite(t *testing.T) {
	line := serve(t, func(w http.ResponseWriter, r *http.Request) {},
		httptest.NewRequest(http.MethodGet, "/x", nil))

	if line["status"] != float64(200) {
		t.Errorf("status = %v, want 200", line["status"])
	}
}

func TestRequestsFirstStatusWins(t *testing.T) {
	line := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.WriteHeader(http.StatusInternalServerError)
	}, httptest.NewRequest(http.MethodGet, "/x", nil))

	if line["status"] != float64(202) {
		t.Errorf("status = %v, want 202", line["status"])
	}
}

func TestRequestsLevels(t *testing.T) {
	tests := []struct {
		status int
		level  string
	}{
		{http.StatusOK, "INFO"},
		{http.StatusFound, "INFO"},
		{http.StatusNotFound, "WARN"},
		{http.StatusInternalServerError, "ERROR"},
	}
	for _, tt := range tests {
		line := serve(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
		}, httptest.NewRequest(http.MethodGet, "/x", nil))

		if line["level"] != tt.level {
			t.Errorf("status %d: level = %v, want %s", tt.status, line["level"], tt.level)
		}
	}
}

func TestRequestsNoQuery(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "production", false)
	h := Requests(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x?token=secret", nil))

	if strings.Contains(buf.String(), "secret") {
		t.Errorf("log output contains the query string: %s", buf.String())
	}
}

func TestRequestsIP(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"forwarded for", "1.2.3.4, 5.6.7.8", "1.2.3.4"},
		{"single forwarded value", "9.9.9.9", "9.9.9.9"},
		{"remote addr", "", "192.0.2.1:1234"}, // httptest.NewRequest default
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("X-Forwarded-For", tt.header)
			}
			line := serve(t, func(w http.ResponseWriter, r *http.Request) {}, req)

			if line["ip"] != tt.want {
				t.Errorf("ip = %v, want %s", line["ip"], tt.want)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "production", false)
	h := Recover(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	want := `{"statusCode":500,"message":"Внутренняя ошибка сервера"}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	line := logLine(t, &buf)
	if line["msg"] != "panic" {
		t.Errorf("msg = %v, want panic", line["msg"])
	}
	if line["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", line["level"])
	}
	if line["panic"] != "boom" {
		t.Errorf("panic = %v, want boom", line["panic"])
	}
	if stack, _ := line["stack"].(string); stack == "" {
		t.Error("stack is empty")
	}
}

func TestRecoverPassesThrough(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "production", false)
	h := Recover(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
	if buf.Len() != 0 {
		t.Errorf("unexpected log output: %s", buf.String())
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		debug     bool
		wantJSON  bool
		wantDebug bool
	}{
		{"production json", "production", false, true, false},
		{"development text", "development", false, false, false},
		{"debug level", "production", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, tt.env, tt.debug)
			log.Debug("dbg")
			log.Info("inf")

			out := buf.String()
			if got := strings.HasPrefix(out, "{"); got != tt.wantJSON {
				t.Errorf("json = %v, want %v: %s", got, tt.wantJSON, out)
			}
			if got := strings.Contains(out, "dbg"); got != tt.wantDebug {
				t.Errorf("debug line present = %v, want %v: %s", got, tt.wantDebug, out)
			}
			if !strings.Contains(out, "inf") {
				t.Errorf("info line missing: %s", out)
			}
		})
	}
}
