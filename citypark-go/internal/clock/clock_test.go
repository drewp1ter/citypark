package clock

import (
	"testing"
	"time"
)

func TestFixed(t *testing.T) {
	want := time.Date(2024, 5, 24, 12, 30, 0, 0, time.UTC)
	c := Fixed(want)
	if got := c.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
	if got := c.Now(); !got.Equal(want) {
		t.Fatalf("second Now() = %v, want %v", got, want)
	}
}

func TestSystemUsesLocation(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*3600)
	before := time.Now()
	got := System(loc).Now()
	after := time.Now()

	if got.Location() != loc {
		t.Fatalf("Location() = %v, want %v", got.Location(), loc)
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, want between %v and %v", got, before, after)
	}
}
