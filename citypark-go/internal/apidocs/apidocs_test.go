package apidocs_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/apidocs"
	"github.com/drewp1ter/citypark-go/internal/app"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const markdownMarker = "<!-- Endpoint sections start below this line. -->"

// routerRoutes returns the "METHOD path" pairs of the real router without the /api/v1 prefix.
func routerRoutes(t *testing.T) map[string]bool {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(cfg, testutil.NewDB(t), log, clock.Fixed(time.Date(2026, 10, 13, 12, 0, 0, 0, cfg.Timezone)))

	routes := map[string]bool{}
	walk := func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		path := strings.TrimPrefix(route, "/api/v1")
		if method == http.MethodOptions || path == "/openapi.yaml" || path == "/docs" || strings.HasPrefix(path, "/uploads/") {
			return nil
		}
		routes[method+" "+path] = true
		return nil
	}
	if err := chi.Walk(a.Handler.(chi.Routes), walk); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return routes
}

// diff fails the test when the two sets differ and prints what is missing on each side.
func diff(t *testing.T, wantName string, want map[string]bool, gotName string, got map[string]bool) {
	t.Helper()
	var missingGot, missingWant []string
	for k := range want {
		if !got[k] {
			missingGot = append(missingGot, k)
		}
	}
	for k := range got {
		if !want[k] {
			missingWant = append(missingWant, k)
		}
	}
	sort.Strings(missingGot)
	sort.Strings(missingWant)
	if len(missingGot) > 0 {
		t.Errorf("in %s but missing from %s: %v", wantName, gotName, missingGot)
	}
	if len(missingWant) > 0 {
		t.Errorf("in %s but missing from %s: %v", gotName, wantName, missingWant)
	}
}

func TestSpecCoversAllRoutes(t *testing.T) {
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}

	methods := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
	inSpec := map[string]bool{}
	inPaths := false
	path := ""
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, " ") && line != "" {
			inPaths = strings.TrimRight(line, " ") == "paths:"
			path = ""
			continue
		}
		if !inPaths {
			continue
		}
		switch {
		case strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			path = strings.TrimSuffix(strings.TrimPrefix(line, "  "), ":")
		case strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") && path != "":
			key := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if methods[key] && strings.HasSuffix(line, ":") {
				inSpec[strings.ToUpper(key)+" "+path] = true
			}
		}
	}
	if len(inSpec) == 0 {
		t.Fatal("no operations found in openapi.yaml")
	}

	diff(t, "router", routerRoutes(t), "openapi.yaml", inSpec)
}

func TestAPIMarkdownCoversAllRoutes(t *testing.T) {
	data, err := os.ReadFile("../../docs/API.md")
	if err != nil {
		t.Fatal(err)
	}
	_, sections, ok := strings.Cut(string(data), markdownMarker)
	if !ok {
		t.Fatalf("docs/API.md has no %q line", markdownMarker)
	}

	inMarkdown := map[string]bool{}
	for _, line := range strings.Split(sections, "\n") {
		if rest, ok := strings.CutPrefix(line, "#### "); ok {
			inMarkdown[strings.TrimSpace(rest)] = true
		}
	}
	if len(inMarkdown) == 0 {
		t.Fatal("no endpoint sections found in docs/API.md")
	}

	diff(t, "router", routerRoutes(t), "docs/API.md", inMarkdown)
}

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	apidocs.Routes(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSpecServed(t *testing.T) {
	rec := serve(t, "/openapi.yaml")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/yaml" {
		t.Errorf("Content-Type = %q, want application/yaml", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "openapi: 3.0") {
		t.Errorf("body starts with %q", rec.Body.String()[:20])
	}
}

func TestDocsPage(t *testing.T) {
	rec := serve(t, "/docs")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "openapi.yaml") {
		t.Error("page does not point at openapi.yaml")
	}
}
