// Package fetch — streaming extraction.
package fetch

import (
	"io"
	"strings"

	"github.com/sirerun/saka/types"
	"golang.org/x/net/html"
)

const streamChunkSize = 900 // ~900 chars per chunk: good granularity for LLM context

// ExtractStream parses HTML and streams extracted text chunks over the
// returned channel as they are discovered. The channel closes when done.
// The final, complete page is delivered via the done channel (or an error).
//
// NOTE (source-chat bug, preserved verbatim): this sets the unexported
// types.Page.text field from package fetch, and calls collectAllText,
// which is never defined anywhere in the source chat. Neither compiles
// as given — see NOTES.md.
func ExtractStream(rawURL string, body io.Reader) (chunks <-chan types.Chunk, done <-chan *types.Page, errc <-chan error) {
	ch := make(chan types.Chunk, 16)
	doneCh := make(chan *types.Page, 1)
	errCh := make(chan error, 1)

	go func() {
		// Only ch is ranged over by consumers; doneCh/errCh each receive
		// exactly one value before return and are read at most once via
		// select, so they must stay unclosed — closing an unwritten
		// buffered channel makes it spuriously receive-ready with the
		// zero value, racing the real send in the consumer's select.
		defer close(ch)

		doc, err := html.Parse(body)
		if err != nil {
			errCh <- err
			return
		}

		title := findMeta(doc, "og:title")
		if title == "" {
			title = findTag(doc, "title")
		}
		published := parseTime(findMeta(doc, "article:published_time"))
		best := findBestContainer(doc)

		seq := 0
		var buf strings.Builder

		flush := func() {
			if buf.Len() == 0 {
				return
			}
			ch <- types.Chunk{Text: buf.String(), Seq: seq}
			seq++
			buf.Reset()
		}

		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if skipTags[n.Data] {
					return
				}
				switch n.Data {
				case "p", "h1", "h2", "h3", "li", "blockquote":
					// paragraph boundary: flush what we have
					if buf.Len() >= streamChunkSize {
						flush()
					}
				}
			}
			if n.Type == html.TextNode {
				buf.WriteString(n.Data)
				if buf.Len() >= streamChunkSize {
					flush()
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(best)
		flush()

		page := &types.Page{
			URL:         rawURL,
			Title:       title,
			PublishedAt: published,
			// full text for caching (source chat sets the unexported
			// `text` field here, which does not compile from this
			// package — see NOTES.md):
			// text: normalize(collectAllText(best)),
			Text: normalize(collectText2(best)),
		}
		doneCh <- page
	}()
	return ch, doneCh, errCh
}

// collectText2 is a stand-in for the source chat's undefined
// collectAllText helper, added only so this file compiles; it reuses the
// existing collectText walker from extract.go.
func collectText2(n *html.Node) string {
	var sb strings.Builder
	collectText(n, &sb)
	return sb.String()
}
