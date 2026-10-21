package validate

import "strings"

var phoneSeparators = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

// NormalizePhone converts a Russian mobile or landline number to "+7XXXXXXXXXX".
// ok is false when the input is not a valid Russian number.
func NormalizePhone(s string) (phone string, ok bool) {
	s = phoneSeparators.Replace(s)

	var digits string
	switch {
	case strings.HasPrefix(s, "+7"):
		digits = s[2:]
	case strings.HasPrefix(s, "7"), strings.HasPrefix(s, "8"):
		digits = s[1:]
	default:
		return "", false
	}

	if len(digits) != 10 {
		return "", false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	return "+7" + digits, true
}
