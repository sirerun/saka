// Package fetch — persistent disk cache (content-addressed JSON files).
package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirerun/saka/types"
)

type DiskCache struct {
	dir string
	ttl time.Duration
}

func NewDiskCache(dir string, ttl time.Duration) (*DiskCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("diskcache: mkdir: %w", err)
	}
	return &DiskCache{dir: dir, ttl: ttl}, nil
}

func (c *DiskCache) key(rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	// shard into 2-char subdirs to keep directories small:
	name := hex.EncodeToString(h[:])
	return filepath.Join(c.dir, name[:2], name+".json")
}

func (c *DiskCache) Get(rawURL string) (*types.Page, bool) {
	path := c.key(rawURL)
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > c.ttl {
		if err == nil {
			os.Remove(path) // lazy expiry cleanup
		}
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var page types.Page
	if err := json.Unmarshal(b, &page); err != nil {
		os.Remove(path) // corrupt entry — drop it
		return nil, false
	}
	return &page, true
}

func (c *DiskCache) Put(rawURL string, page *types.Page) {
	b, err := json.Marshal(page)
	if err != nil {
		return
	}
	path := c.key(rawURL)
	os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, path) // atomic write
	}
}

// GC removes expired entries; call periodically from the server,
// or opportunistically on startup.
func (c *DiskCache) GC() (removed int) {
	filepath.WalkDir(c.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		if info, err := d.Info(); err == nil && time.Since(info.ModTime()) > c.ttl {
			os.Remove(path)
			removed++
		}
		return nil
	})
	return removed
}
