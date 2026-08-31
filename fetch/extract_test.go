package fetch

import (
	"strings"
	"testing"
)

var articleFixture = `<!DOCTYPE html>
<html><head>
<title>My Article</title>
<meta property="og:title" content="OG Title">
<meta property="article:published_time" content="2025-01-15T10:00:00Z">
</head><body>
<nav>Home | About | Login</nav>
<aside>Related links everywhere</aside>
<article>
  <h1>OG Title</h1>
  <p>` + strings.Repeat("This is the main article content. ", 30) + `</p>
  <p>` + strings.Repeat("Another substantive paragraph here. ", 25) + `</p>
</article>
<footer>© 2025 Example Corp</footer>
</body></html>`

func TestExtractArticle(t *testing.T) {
	page, err := Extract("https://example.com/a", strings.NewReader(articleFixture))
	if err != nil {
		t.Fatal(err)
	}
	if page.Title != "OG Title" {
		t.Errorf("title = %q", page.Title)
	}
	if page.PublishedAt.IsZero() || page.PublishedAt.Year() != 2025 {
		t.Errorf("published_at not parsed: %v", page.PublishedAt)
	}
	if !strings.Contains(page.Text, "main article content") {
		t.Errorf("body text missing:\n%s", page.Text)
	}
	if strings.Contains(page.Text, "Related links") {
		t.Errorf("aside leaked into extracted text")
	}
	if strings.Contains(page.Text, "© 2025") {
		t.Errorf("footer leaked into extracted text")
	}
	if len(page.Text) < 500 {
		t.Errorf("extracted text too short: %d bytes", len(page.Text))
	}
}

func TestExtractSkipsScripts(t *testing.T) {
	html := `<html><body><article><p>Real words here.</p><script>var x=1;garbage();</script></article></body></html>`
	page, err := Extract("http://x", strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page.Text, "garbage") {
		t.Error("script content leaked")
	}
	if !strings.Contains(page.Text, "Real words here.") {
		t.Error("paragraph lost")
	}
}
