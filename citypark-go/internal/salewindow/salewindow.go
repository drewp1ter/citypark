// Package salewindow models the time and weekday limits on when a category or
// product can be ordered, as stored in the iiko additionalInfo field.
package salewindow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const minutesPerDay = 24 * 60

var (
	daysPattern = regexp.MustCompile(`^[01]{1,7}$`)
	dayNames    = [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
)

// Window limits when a category or product can be ordered.
type Window struct {
	Start *int   // minutes from midnight; nil = no time limit
	End   *int   // minutes from midnight; nil = no time limit
	Days  string // "" = every day, else 7 chars of '0'/'1', Monday first
}

// ParseClock parses "H:MM" or "HH:MM" (surrounding spaces allowed) into minutes from midnight.
// Hours 0..24, minutes 0..59, result at most 1440. "9:00" -> 540, "21:30" -> 1290.
func ParseClock(s string) (minutes int, ok bool) {
	h, m, found := strings.Cut(strings.TrimSpace(s), ":")
	if !found || len(h) < 1 || len(h) > 2 || len(m) != 2 {
		return 0, false
	}
	hours, ok := digits(h)
	if !ok {
		return 0, false
	}
	mins, ok := digits(m)
	if !ok {
		return 0, false
	}
	if hours > 24 || mins > 59 {
		return 0, false
	}
	total := hours*60 + mins
	if total > minutesPerDay {
		return 0, false
	}
	return total, true
}

// digits converts a string of ASCII digits to an int.
func digits(s string) (int, bool) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// Parse reads the iiko additionalInfo value: a JSON array [startTime, endTime, daysOfWeek].
// It never fails: anything it cannot read means "no restriction".
func Parse(additionalInfo string) Window {
	var parts []json.RawMessage
	if err := json.Unmarshal([]byte(additionalInfo), &parts); err != nil {
		return Window{}
	}

	var w Window
	if len(parts) >= 2 {
		start, okStart := parseClockElement(parts[0])
		end, okEnd := parseClockElement(parts[1])
		if okStart && okEnd {
			w.Start, w.End = &start, &end
		}
	}
	if len(parts) >= 3 {
		w.Days = parseDays(parts[2])
	}
	return w
}

func parseClockElement(raw json.RawMessage) (int, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, false
	}
	return ParseClock(s)
}

// parseDays accepts the days element as a JSON string or number and returns
// the 7-char mask, or "" when it is not a valid mask.
func parseDays(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, `"`) {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		text = strings.TrimSpace(s)
	}
	if !daysPattern.MatchString(text) {
		return ""
	}
	return text + strings.Repeat("0", 7-len(text))
}

// OpenAtTime reports whether the time of day of t is inside the window.
func (w Window) OpenAtTime(t time.Time) bool {
	if w.Start == nil || w.End == nil {
		return true
	}
	m := t.Hour()*60 + t.Minute()
	return *w.Start <= m && m < *w.End
}

// OpenOnDay reports whether the weekday of t is allowed.
func (w Window) OpenOnDay(t time.Time) bool {
	if w.Days == "" {
		return true
	}
	return w.Days[(int(t.Weekday())+6)%7] == '1'
}

// StartString formats Start as "9:00", or "" when Start is nil.
func (w Window) StartString() string { return formatMinutes(w.Start) }

// EndString formats End as "21:00", or "" when End is nil.
func (w Window) EndString() string { return formatMinutes(w.End) }

func formatMinutes(m *int) string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("%d:%02d", *m/60, *m%60)
}

// DaysString lists the allowed days like "Пн, Вт, Ср", or "" when Days is "".
func (w Window) DaysString() string {
	var names []string
	for i, c := range w.Days {
		if c == '1' {
			names = append(names, dayNames[i])
		}
	}
	return strings.Join(names, ", ")
}
