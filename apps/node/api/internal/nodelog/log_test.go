package nodelog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanRemovesOnlyExpiredLogFiles(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "one-proxy-node-20260616-10.log")
	newPath := filepath.Join(dir, "one-proxy-node-20260616-11.log")
	otherPath := filepath.Join(dir, "other.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherPath, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-73*time.Hour), now.Add(-73*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(otherPath, now.Add(-73*time.Hour), now.Add(-73*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := clean(dir, 72*time.Hour, 1024, now, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file missing err=%v", err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("non-node log file missing err=%v", err)
	}
}

func TestRotatingWriterUsesHourlyFiles(t *testing.T) {
	dir := t.TempDir()
	writer := &rotatingWriter{dir: dir}
	first := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	if err := writer.rotate(first); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.file.Write([]byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := writer.rotate(second); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.file.Write([]byte("second")); err != nil {
		t.Fatal(err)
	}
	if err := writer.file.Close(); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := os.ReadFile(filepath.Join(dir, "one-proxy-node-20260616-10.log"))
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(dir, "one-proxy-node-20260616-11.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != "first" || string(secondBytes) != "second" {
		t.Fatalf("rotated files = %q %q", firstBytes, secondBytes)
	}
}

func TestCleanEnforcesDiskLimitByRemovingOldestInactiveLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	oldPath := filepath.Join(dir, "one-proxy-node-20260616-10.log")
	midPath := filepath.Join(dir, "one-proxy-node-20260616-11.log")
	activePath := filepath.Join(dir, "one-proxy-node-20260616-12.log")
	for _, item := range []struct {
		path string
		age  time.Duration
		body string
	}{
		{oldPath, 3 * time.Hour, "old-"},
		{midPath, 2 * time.Hour, "mid-"},
		{activePath, time.Hour, "active"},
	} {
		if err := os.WriteFile(item.path, []byte(item.body), 0o644); err != nil {
			t.Fatal(err)
		}
		ts := now.Add(-item.age)
		if err := os.Chtimes(item.path, ts, ts); err != nil {
			t.Fatal(err)
		}
	}

	if err := clean(dir, 72*time.Hour, 10, now, activePath); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists err=%v", err)
	}
	if _, err := os.Stat(midPath); err != nil {
		t.Fatalf("mid file missing err=%v", err)
	}
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("active file missing err=%v", err)
	}
}
