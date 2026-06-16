package nodelog

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Dir       string
	Retention time.Duration
}

type rotatingWriter struct {
	mu   sync.Mutex
	dir  string
	slot string
	file *os.File
}

func Configure(cfg Config) error {
	if cfg.Dir == "" {
		return nil
	}
	if cfg.Retention <= 0 {
		cfg.Retention = 72 * time.Hour
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return err
	}
	if err := clean(cfg.Dir, cfg.Retention, time.Now()); err != nil {
		return err
	}
	writer := &rotatingWriter{dir: cfg.Dir}
	if err := writer.rotate(time.Now()); err != nil {
		return err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, writer))
	go runCleaner(cfg.Dir, cfg.Retention)
	return nil
}

func (w *rotatingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotate(time.Now()); err != nil {
		return 0, err
	}
	return w.file.Write(data)
}

func (w *rotatingWriter) rotate(now time.Time) error {
	slot := now.UTC().Format("20060102-15")
	if w.file != nil && w.slot == slot {
		return nil
	}
	path := filepath.Join(w.dir, "one-proxy-node-"+slot+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = file
	w.slot = slot
	return nil
}

func runCleaner(dir string, retention time.Duration) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for now := range ticker.C {
		_ = clean(dir, retention, now)
	}
}

func clean(dir string, retention time.Duration, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := now.Add(-retention)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}
