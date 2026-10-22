package richtext

import (
	"sync"
	"testing"
)

func TestSanitize(t *testing.T) {
	const vk = `<a href="https://vk.com/cityparkvip" rel="noreferrer noopener" target="_blank">vk</a>`
	tests := []struct {
		name, in, article, short string
	}{
		{"formatting", `<p>Привет <strong>мир</strong></p>`, `<p>Привет <strong>мир</strong></p>`, `<p>Привет <strong>мир</strong></p>`},
		{"em and br", `<p>a<br>b <em>c</em></p>`, `<p>a<br>b <em>c</em></p>`, `<p>a<br>b <em>c</em></p>`},
		{"heading", `<h2>Меню</h2><p>x</p>`, `<h2>Меню</h2><p>x</p>`, `Меню<p>x</p>`},
		{"h3", `<h3>Т</h3><p>x</p>`, `<h3>Т</h3><p>x</p>`, `Т<p>x</p>`},
		{"h1 dropped", `<h1>Заголовок</h1>`, `Заголовок`, `Заголовок`},
		{"blockquote", `<blockquote>q</blockquote>`, `<blockquote>q</blockquote>`, `q`},
		{"attributes", `<p onclick="x()" style="color:red" class="c" id="i">a</p>`, `<p>a</p>`, `<p>a</p>`},
		{"script", `<script>alert(1)</script><p>a</p>`, `<p>a</p>`, `<p>a</p>`},
		{"style", `<style>p{color:red}</style><p>a</p>`, `<p>a</p>`, `<p>a</p>`},
		{"iframe", `<iframe src="https://x.ru">hidden</iframe><p>a</p>`, `<p>a</p>`, `<p>a</p>`},
		{"javascript link", `<a href="javascript:alert(1)">x</a>`, `x`, `x`},
		{"external link", `<a href="https://vk.com/cityparkvip">vk</a>`, vk, vk},
		{"link attributes", `<a href="https://vk.com/cityparkvip" class="c" onclick="x()">vk</a>`, vk, vk},
		{"relative link", `<a href="/rooms">номера</a>`, `<a href="/rooms" rel="noreferrer">номера</a>`, `<a href="/rooms" rel="noreferrer">номера</a>`},
		{"tel", `<a href="tel:+79183119791">tel</a>`, `<a href="tel:+79183119791" rel="noreferrer">tel</a>`, `<a href="tel:+79183119791" rel="noreferrer">tel</a>`},
		{"mailto", `<a href="mailto:a@b.ru">m</a>`, `<a href="mailto:a@b.ru" rel="noreferrer">m</a>`, `<a href="mailto:a@b.ru" rel="noreferrer">m</a>`},
		{"lists", `<ul><li>a</li></ul><ol><li>b</li></ol>`, `<ul><li>a</li></ul><ol><li>b</li></ol>`, `<ul><li>a</li></ul><ol><li>b</li></ol>`},
		{"b is not strong", `<p><b>x</b></p>`, `<p>x</p>`, `<p>x</p>`},
		{"img", `<img src=x onerror=alert(1)>`, ``, ``},
		{"empty", ``, ``, ``},
		{"empty p", `<p></p>`, ``, ``},
		{"p br", `<p><br></p>`, ``, ``},
		{"spaces", `  `, ``, ``},
		{"nbsp only", `<p>&nbsp;</p>`, ``, ``},
		{"empty list", `<ul><li></li></ul>`, ``, ``},
		{"entities", `<p>a &amp; b</p>`, `<p>a &amp; b</p>`, `<p>a &amp; b</p>`},
		{"trimmed", "  <p>a</p>\n", `<p>a</p>`, `<p>a</p>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(Article, tt.in); got != tt.article {
				t.Errorf("Article: got %q, want %q", got, tt.article)
			}
			if got := Sanitize(Short, tt.in); got != tt.short {
				t.Errorf("Short: got %q, want %q", got, tt.short)
			}
		})
	}
}

func TestSanitizeConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := Sanitize(Article, `<h2>a</h2>`); got != `<h2>a</h2>` {
				t.Errorf("got %q", got)
			}
		}()
	}
	wg.Wait()
}

func TestTextLength(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{`<p>При&nbsp;вет</p>`, 7},
		{``, 0},
		{`<ul><li>ab</li></ul>`, 2},
		{`<p>a &amp; b</p>`, 5},
	}
	for _, tt := range tests {
		if got := TextLength(tt.in); got != tt.want {
			t.Errorf("TextLength(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
