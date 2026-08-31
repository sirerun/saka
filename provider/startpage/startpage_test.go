package startpage

import (
	"strings"
	"testing"
)

const fixture = `<!DOCTYPE html>
<html><body>
<div class="w-gl">
  <a class="w-gl__result-title" href="https://example.com/one"><h2>First</h2></a>
  <p class="w-gl__description">Snippet one</p>
  <a class="w-gl__result-title" href="https://example.com/two"><h2>Second</h2></a>
  <p class="w-gl__description">Snippet two</p>
</div>
</body></html>`

const challengeFixture = `<html><body><div class="captcha-container">Please verify</div></body></html>`

func TestParseResults(t *testing.T) {
	results, err := parseResults(strings.NewReader(fixture), 10)
	if err != nil || len(results) != 2 {
		t.Fatalf("got %d results, err %v", len(results), err)
	}
	if results[0].URL != "https://example.com/one" || results[0].Snippet != "Snippet one" {
		t.Errorf("bad first result: %+v", results[0])
	}
}

func TestParseResultsChallengeIsEmpty(t *testing.T) {
	results, _ := parseResults(strings.NewReader(challengeFixture), 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for challenge page, got %d", len(results))
	}
}
