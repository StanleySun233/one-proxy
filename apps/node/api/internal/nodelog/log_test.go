package nodelog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanRemovesOnlyExpiredLogFiles(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.log")
	newPath := filepath.Join(dir, "new.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-73*time.Hour), now.Add(-73*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := clean(dir, 72*time.Hour, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file missing err=%v", err)
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
