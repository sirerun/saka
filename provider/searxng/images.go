package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	types "github.com/sirerun/saka/types"
)

// ImagesProvider queries the same self-hosted SearXNG instance as Provider
// but requests categories=images and parses SearXNG's image-result JSON
// shape instead of the general-web shape. It embeds *Provider to reuse its
// HTTP client and base-URL config rather than duplicating them.
//
// This provider does NOT decide which vertical it belongs to. Assigning it
// to the "images" vertical is types.ProviderConfig.Vertical's job (ADR 003
// and its 2026-08-29 addendum) -- a future reader must not wire
// "searxng-images" into the general web chain by hand; the vertical is
// config, not code here.
type ImagesProvider struct {
	*Provider
}

// NewImages builds an ImagesProvider sharing Provider's HTTP client and
// base-URL construction.
func NewImages(baseURL string) *ImagesProvider {
	return &ImagesProvider{Provider: New(baseURL)}
}

func (p *ImagesProvider) Name() string { return "searxng-images" }

func init() {
	if err := types.Register("searxng-images", newImagesFromConfig); err != nil {
		panic(err)
	}
}

func newImagesFromConfig(cfg types.ProviderConfig) (types.Provider, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("saka: searxng-images provider requires url")
	}
	return NewImages(cfg.URL), nil
}

// searxImagesResponse models the subset of SearXNG's categories=images JSON
// we need -- distinct from searxResponse's general-web shape.
type searxImagesResponse struct {
	Results []struct {
		Title        string `json:"title"`
		ImgSrc       string `json:"img_src"`
		ThumbnailSrc string `json:"thumbnail_src,omitempty"`
		Engine       string `json:"engine,omitempty"`
		Resolution   string `json:"resolution,omitempty"` // e.g. "1920x1080"
	} `json:"results"`
}

func (p *ImagesProvider) Search(ctx context.Context, q types.Query) ([]types.Result, error) {
	params := url.Values{
		"q":          {q.Text},
		"format":     {"json"},
		"language":   {langOrDefault(q.Region)},
		"categories": {"images"},
	}
	if q.MaxResults > 0 {
		params.Set("pageno", "1")
	}
	if q.Site != "" {
		params.Set("q", q.Text+" site:"+q.Site)
	}
	if q.SafeSearch {
		params.Set("safesearch", "1")
	}

	endpoint := p.baseURL + "/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	// SearXNG requires a browser-like UA or it may return 403.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; saka/0.1)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng-images: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusForbidden:
		return nil, &types.RateLimitError{Provider: "searxng-images", RetryAfter: 30 * time.Second}
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("searxng-images: status %d", resp.StatusCode)
	}

	var sr searxImagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("searxng-images: decode: %w", err)
	}

	out := make([]types.Result, 0, len(sr.Results))
	for i, r := range sr.Results {
		if i >= q.MaxResults {
			break
		}
		width, height := parseResolution(r.Resolution)
		out = append(out, types.Result{
			Title:        r.Title,
			URL:          r.ImgSrc,
			ThumbnailURL: r.ThumbnailSrc,
			Source:       "searxng-images",
			Position:     i + 1,
			Width:        width,
			Height:       height,
		})
	}
	return out, nil
}

// parseResolution parses a SearXNG resolution string like "1920x1080" into
// width and height. A malformed or absent resolution leaves both at zero
// rather than failing the whole result.
func parseResolution(res string) (width, height int) {
	parts := strings.SplitN(res, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0
	}
	return w, h
}
