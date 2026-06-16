package responsecache

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCanStoreOnlySafeCompleteResponses(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	header := http.Header{"Content-Type": []string{"application/json"}}
	if !CanStore(req, http.StatusOK, header, 2) {
		t.Fatal("expected cacheable response")
	}
	post, err := http.NewRequest(http.MethodPost, "http://example.test/data", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if CanStore(post, http.StatusOK, header, 2) {
		t.Fatal("post should not be cacheable")
	}
	noStore := header.Clone()
	noStore.Set("Cache-Control", "no-store")
	if CanStore(req, http.StatusOK, noStore, 2) {
		t.Fatal("no-store should not be cacheable")
	}
	withCookie := header.Clone()
	withCookie.Set("Set-Cookie", "sid=1")
	if CanStore(req, http.StatusOK, withCookie, 2) {
		t.Fatal("set-cookie should not be cacheable")
	}
	withVary := header.Clone()
	withVary.Set("Vary", "Accept-Encoding")
	if CanStore(req, http.StatusOK, withVary, 2) {
		t.Fatal("vary should not be cacheable")
	}
}

func TestCacheClearsDiskOnStartupAndExpiresEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.cache"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := New(Config{Dir: dir, TTL: 20 * time.Millisecond, MemoryMaxBytes: 1024, DiskMaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.cache")); !os.IsNotExist(err) {
		t.Fatalf("old cache file still exists err=%v", err)
	}
	key := "abc"
	if err := cache.Set(key, Entry{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: []byte("fresh")}); err != nil {
		t.Fatal(err)
	}
	entry, err := cache.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if string(entry.Body) != "fresh" {
		t.Fatalf("body = %q", entry.Body)
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := cache.Get(key); err != ErrMiss {
		t.Fatalf("expected miss after ttl, got %v", err)
	}
}

func TestKeyIncludesIdentityHeaders(t *testing.T) {
	first, err := http.NewRequest(http.MethodGet, "http://example.test/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := http.NewRequest(http.MethodGet, "http://example.test/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	first.Header.Set("Authorization", "Bearer one")
	second.Header.Set("Authorization", "Bearer two")
	if Key(first, nil) == Key(second, nil) {
		t.Fatal("identity headers should affect cache key")
	}
}

func TestKeyIgnoresClientRefreshHeaders(t *testing.T) {
	first, err := http.NewRequest(http.MethodGet, "http://example.test/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := http.NewRequest(http.MethodGet, "http://example.test/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	second.Header.Set("Cache-Control", "no-cache")
	second.Header.Set("Pragma", "no-cache")
	if Key(first, nil) != Key(second, nil) {
		t.Fatal("refresh headers should not affect cache key")
	}
}
