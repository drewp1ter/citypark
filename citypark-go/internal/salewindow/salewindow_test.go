package salewindow

import (
	"testing"
	"time"
)

func intPtr(v int) *int { return &v }

func ptrEq(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestParseClock(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"9:00", 540, true},
		{" 21:30 ", 1290, true},
		{"24:00", 1440, true},
		{"0:00", 0, true},
		{"", 0, false},
		{"9", 0, false},
		{"25:00", 0, false},
		{"9:60", 0, false},
		{"a:b", 0, false},
		{"24:01", 0, false},
		{"9:5", 0, false},
		{"+9:00", 0, false},
		{"123:00", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := ParseClock(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ParseClock(%q) = %d, %v; want %d, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		start *int
		end   *int
		days  string
	}{
		{"empty", ``, nil, nil, ""},
		{"not json", `not json`, nil, nil, ""},
		{"not array", `{"a": 1}`, nil, nil, ""},
		{"days as number", `["9:00", "21:00", 1111100]`, intPtr(540), intPtr(1260), "1111100"},
		{"days as string", `["9:00", "21:00", "0000011"]`, intPtr(540), intPtr(1260), "0000011"},
		{"only start", `["9:00"]`, nil, nil, ""},
		{"empty times, short days", `["", "", "11111"]`, nil, nil, "1111100"},
		{"bad days", `["9:00", "21:00", "abc"]`, intPtr(540), intPtr(1260), ""},
		{"one bad time", `["9:00", "x", "1111111"]`, nil, nil, "1111111"},
		{"time not a string", `[900, 2100]`, nil, nil, ""},
		{"too many day chars", `["9:00", "21:00", "11111111"]`, intPtr(540), intPtr(1260), ""},
		{"days null", `["9:00", "21:00", null]`, intPtr(540), intPtr(1260), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Parse(tt.in)
			if !ptrEq(w.Start, tt.start) || !ptrEq(w.End, tt.end) || w.Days != tt.days {
				t.Fatalf("Parse(%q) = {%v %v %q}", tt.in, w.Start, w.End, w.Days)
			}
		})
	}
}

func TestOpenAtTime(t *testing.T) {
	w := Window{Start: intPtr(540), End: intPtr(1260)}
	day := func(h, m int) time.Time { return time.Date(2024, 5, 24, h, m, 0, 0, time.UTC) }

	tests := []struct {
		name string
		w    Window
		t    time.Time
		want bool
	}{
		{"before open", w, day(8, 59), false},
		{"at open", w, day(9, 0), true},
		{"before close", w, day(20, 59), true},
		{"at close", w, day(21, 0), false},
		{"zero window", Window{}, day(3, 0), true},
		{"only start set", Window{Start: intPtr(540)}, day(3, 0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.OpenAtTime(tt.t); got != tt.want {
				t.Fatalf("OpenAtTime(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestOpenOnDay(t *testing.T) {
	w := Window{Days: "1111100"}
	tests := []struct {
		name string
		w    Window
		t    time.Time
		want bool
	}{
		{"friday", w, time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC), true},
		{"saturday", w, time.Date(2024, 5, 25, 12, 0, 0, 0, time.UTC), false},
		{"sunday", w, time.Date(2024, 5, 26, 12, 0, 0, 0, time.UTC), false},
		{"monday", w, time.Date(2024, 5, 27, 12, 0, 0, 0, time.UTC), true},
		{"zero window", Window{}, time.Date(2024, 5, 26, 12, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.OpenOnDay(tt.t); got != tt.want {
				t.Fatalf("OpenOnDay(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestStrings(t *testing.T) {
	w := Window{Start: intPtr(540), End: intPtr(1260), Days: "1111100"}
	if got := w.StartString(); got != "9:00" {
		t.Errorf("StartString() = %q, want %q", got, "9:00")
	}
	if got := w.EndString(); got != "21:00" {
		t.Errorf("EndString() = %q, want %q", got, "21:00")
	}
	if got := w.DaysString(); got != "Пн, Вт, Ср, Чт, Пт" {
		t.Errorf("DaysString() = %q, want %q", got, "Пн, Вт, Ср, Чт, Пт")
	}

	var zero Window
	if zero.StartString() != "" || zero.EndString() != "" || zero.DaysString() != "" {
		t.Errorf("zero window strings = %q %q %q, want all empty",
			zero.StartString(), zero.EndString(), zero.DaysString())
	}

	odd := Window{Start: intPtr(65), End: intPtr(1440)}
	if got := odd.StartString(); got != "1:05" {
		t.Errorf("StartString() = %q, want %q", got, "1:05")
	}
	if got := odd.EndString(); got != "24:00" {
		t.Errorf("EndString() = %q, want %q", got, "24:00")
	}
}
