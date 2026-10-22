# Task 47: Rich text sanitizer

## Goal
Package `richtext` cleans HTML from the admin's editor with two allowlists, *article* and *short*, so the content
endpoints (tasks 48–50) store only safe, expected markup.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (entry "Rich text profile").

## Background
The admin uses a Tiptap editor and sends HTML; the site renders it as HTML. The server is the only place that can
guarantee no `<script>`, `onclick`, `style` or `javascript:` link gets stored, also for hand-made API calls. The admin
toolbar offers exactly the tags of the profile, so what staff see is what the site shows.

This task adds the dependency `github.com/microcosm-cc/bluemonday` (latest release). It is the only new dependency.

## Files to create
- `internal/richtext/richtext.go`, `internal/richtext/richtext_test.go`

## Files to change
- `go.mod`, `go.sum`; `docs/tasks/CONVENTIONS.md`: add bluemonday to "Allowed dependencies".

## Spec
```go
type Profile int

const (
	Article Profile = iota // news body
	Short                  // room descriptions, hotel texts, home page blocks
)

// Sanitize returns the cleaned HTML for the profile. Empty markup ("", "<p></p>", "<p><br></p>", only spaces)
// becomes "". The result is trimmed.
func Sanitize(p Profile, html string) string

// TextLength is the number of characters (runes) of visible text in already sanitized HTML, used for max-length rules.
func TextLength(html string) int
```
Allowed elements:
- both profiles: `p`, `br`, `strong`, `em`, `a`, `ul`, `ol`, `li`;
- `Article` also: `h2`, `h3`, `blockquote`.

Rules (one `*bluemonday.Policy` per profile, built once in `init`-free package-level constructors called from
`Sanitize` through a `sync.OnceValue`; policies are safe for concurrent use):
- `a` keeps only `href`; schemes `http`, `https`, `mailto`, `tel` and relative URLs starting with `/`
  (`AllowRelativeURLs(true)` + `AllowURLSchemes(...)`); every link gets `rel="noopener noreferrer"`; fully
  qualified links get `target="_blank"` (`AddTargetBlankToFullyQualifiedLinks(true)`).
- No other attributes at all (no `class`, `style`, `id`).
- Elements not allowed are removed but their text is kept (`<h1>Заголовок</h1>` → `Заголовок` in Short;
  bluemonday's default), except `script`, `style`, `iframe`, whose content is dropped.
- `b` → keep as text only (do not map it to `strong`; Tiptap emits `strong` already).

`TextLength`: strip tags with `bluemonday.StrictPolicy()`, unescape HTML entities (`html.UnescapeString`),
`utf8.RuneCountInString`.

## Tests to write
Table tests for `Sanitize` (input → expected output, exact strings):
- `<p>Привет <strong>мир</strong></p>` → unchanged, both profiles.
- `<h2>Меню</h2><p>x</p>` → Article unchanged; Short `Меню<p>x</p>`.
- `<blockquote>q</blockquote>` → Article unchanged; Short `q`.
- `<p onclick="x()" style="color:red" class="c">a</p>` → `<p>a</p>`.
- `<script>alert(1)</script><p>a</p>` → `<p>a</p>`.
- `<a href="javascript:alert(1)">x</a>` → `x`.
- `<a href="https://vk.com/cityparkvip">vk</a>` → `<a href="https://vk.com/cityparkvip" rel="noopener noreferrer" target="_blank">vk</a>`
  (check the attribute order bluemonday produces and pin it).
- `<a href="/rooms">номера</a>` → relative link kept with `rel`, no `target`.
- `<a href="tel:+79183119791">tel</a>` → kept.
- `<ul><li>a</li></ul><ol><li>b</li></ol>` → unchanged.
- `<img src=x onerror=alert(1)>` → ``.
- `""`, `<p></p>`, `<p><br></p>`, `  ` → `""`.
- `<p>a &amp; b</p>` → unchanged.

`TextLength`: `<p>При&nbsp;вет</p>` → 7; `""` → 0; `<ul><li>ab</li></ul>` → 2.

## Done when
The commands in CONVENTIONS "Done means" pass and coverage of `internal/richtext` is at least 80%.

## Out of scope
Using the package in endpoints (tasks 48–50), images inside rich text, Markdown.
