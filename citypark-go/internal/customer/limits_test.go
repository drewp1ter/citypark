package customer_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/customer"
)

func TestCustomerAuthLimits(t *testing.T) {
	c := &movingClock{time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
	l := customer.NewLimits(c)
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "192.0.2.1:5000"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	for i := 0; i < 10; i++ {
		if ok, _ := l.SignIn("a@example.com", r); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := l.SignIn("a@example.com", r); ok || wait != 900 {
		t.Fatalf("limit %v %d", ok, wait)
	}
	if ok, _ := l.SignIn("b@example.com", r); !ok {
		t.Fatal("email isolation")
	}
	c.now = c.now.Add(15 * time.Minute)
	if ok, _ := l.SignIn("a@example.com", r); !ok {
		t.Fatal("window boundary")
	}
	for i := 0; i < 3; i++ {
		if !l.Send("a@example.com", r) {
			t.Fatal(i)
		}
	}
	if l.Send("a@example.com", r) {
		t.Fatal("send limit")
	}
	c.now = c.now.Add(time.Hour)
	if !l.Send("a@example.com", r) {
		t.Fatal("send boundary")
	}
	for i := 0; i < 20; i++ {
		if ok, _ := l.Confirm(r); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := l.Confirm(r); ok || wait != 900 {
		t.Fatalf("confirm %v %d", ok, wait)
	}
}
