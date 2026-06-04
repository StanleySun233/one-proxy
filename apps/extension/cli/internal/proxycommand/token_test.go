package proxycommand

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTokenFromEnv(t *testing.T) {
	t.Setenv("ONEPROXY_TEST_TOKEN", " token-value ")
	token, err := ReadToken("ONEPROXY_TEST_TOKEN", "")
	if err != nil {
		t.Fatal(err)
	}
	if token != "token-value" {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestReadTokenFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(" file-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	token, err := ReadToken("ONEPROXY_TEST_TOKEN", path)
	if err != nil {
		t.Fatal(err)
	}
	if token != "file-token" {
		t.Fatalf("unexpected token %q", token)
	}
}
