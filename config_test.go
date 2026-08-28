package saka

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"default ok", DefaultConfig(), false},
		{"no providers", Config{Fetch: DefaultConfig().Fetch}, true},
		{"searxng missing url", Config{Providers: []ProviderConfig{{Name: "searxng"}}}, true},
		{"searxng ok", Config{Providers: []ProviderConfig{{Name: "searxng", URL: "http://localhost:8888"}}}, false},
		{"unknown provider", Config{Providers: []ProviderConfig{{Name: "google"}}}, true},
		{"duplicate", Config{Providers: []ProviderConfig{{Name: "duckduckgo"}, {Name: "duckduckgo"}}}, true},
		{"rps too high", Config{Providers: []ProviderConfig{{Name: "duckduckgo", RPS: 99}}}, true},
	}
	for _, c := range cases {
		if err := c.cfg.Validate(); (err != nil) != c.wantErr {
			t.Errorf("%s: Validate() err = %v, wantErr %v", c.name, err, c.wantErr)
		}
	}
}
