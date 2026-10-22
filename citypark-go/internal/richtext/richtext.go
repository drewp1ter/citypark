// Package richtext cleans HTML from the admin's editor with an allowlist per profile.
package richtext

import (
	"html"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

// Profile selects the set of allowed tags.
type Profile int

const (
	Article Profile = iota // news body
	Short                  // room descriptions, hotel texts, home page blocks
)

var (
	articlePolicy = sync.OnceValue(func() *bluemonday.Policy { return newPolicy(Article) })
	shortPolicy   = sync.OnceValue(func() *bluemonday.Policy { return newPolicy(Short) })
	strictPolicy  = sync.OnceValue(bluemonday.StrictPolicy)
)

func newPolicy(p Profile) *bluemonday.Policy {
	pol := bluemonday.NewPolicy()
	pol.AllowElements("p", "br", "strong", "em", "ul", "ol", "li")
	if p == Article {
		pol.AllowElements("h2", "h3", "blockquote")
	}
	pol.AllowAttrs("href").OnElements("a")
	pol.AllowRelativeURLs(true)
	pol.AllowURLSchemes("http", "https", "mailto", "tel")
	pol.RequireNoFollowOnLinks(false)
	pol.RequireNoReferrerOnLinks(true)
	pol.AddTargetBlankToFullyQualifiedLinks(true)
	pol.SkipElementsContent("script", "style", "iframe")
	return pol
}

// Sanitize returns the cleaned HTML for the profile. Empty markup ("", "<p></p>", "<p><br></p>", only spaces)
// becomes "". The result is trimmed.
func Sanitize(p Profile, markup string) string {
	pol := shortPolicy()
	if p == Article {
		pol = articlePolicy()
	}
	out := strings.TrimSpace(pol.Sanitize(markup))
	if strings.TrimSpace(plainText(out)) == "" {
		return ""
	}
	return out
}

// TextLength is the number of characters (runes) of visible text in already sanitized HTML, used for max-length rules.
func TextLength(markup string) int {
	return utf8.RuneCountInString(plainText(markup))
}

func plainText(markup string) string {
	return html.UnescapeString(strictPolicy().Sanitize(markup))
}
