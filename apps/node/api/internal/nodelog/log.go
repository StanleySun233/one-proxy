package nodelog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Dir           string
	Retention     time.Duration
	MaxDiskBytes  int64
	CleanInterval time.Duration
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
		cfg.Retention = 14 * 24 * time.Hour
	}
	if cfg.MaxDiskBytes <= 0 {
		cfg.MaxDiskBytes = 1024 * 1024 * 1024
	}
	if cfg.CleanInterval <= 0 {
		cfg.CleanInterval = time.Hour
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return err
	}
	writer := &rotatingWriter{dir: cfg.Dir}
	if err := writer.rotate(time.Now()); err != nil {
		return err
	}
	if err := writer.clean(cfg, time.Now()); err != nil {
		return err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, writer))
	go runCleaner(cfg, writer)
	return nil
}

func (w *rotatingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotate(time.Now()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "node log rotate failed err=%v\n", err)
		return len(data), nil
	}
	if _, err := w.file.Write(data); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "node log write failed err=%v\n", err)
		return len(data), nil
	}
	return len(data), nil
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

func (w *rotatingWriter) clean(cfg Config, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return clean(cfg.Dir, cfg.Retention, cfg.MaxDiskBytes, now, w.activePathLocked())
}

func (w *rotatingWriter) activePathLocked() string {
	if w.slot == "" {
		return ""
	}
	return filepath.Join(w.dir, "one-proxy-node-"+w.slot+".log")
}

func runCleaner(cfg Config, writer *rotatingWriter) {
	ticker := time.NewTicker(cfg.CleanInterval)
	defer ticker.Stop()
	for now := range ticker.C {
		if err := writer.clean(cfg, now); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "node log cleanup failed err=%v\n", err)
		}
	}
}

func clean(dir string, retention time.Duration, maxDiskBytes int64, now time.Time, activePath string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := now.Add(-retention)
	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isNodeLogFile(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if path != activePath && info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				continue
			}
		}
		files = append(files, logFile{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}
	enforceDiskLimit(files, maxDiskBytes, activePath)
	return nil
}

type logFile struct {
	path    string
	size    int64
	modTime time.Time
}

func isNodeLogFile(name string) bool {
	return strings.HasPrefix(name, "one-proxy-node-") && strings.HasSuffix(name, ".log")
}

func enforceDiskLimit(files []logFile, maxDiskBytes int64, activePath string) {
	if maxDiskBytes <= 0 {
		return
	}
	var total int64
	for _, file := range files {
		total += file.size
	}
	if total <= maxDiskBytes {
		return
	}
	sort.Slice(files, func(i int, j int) bool {
		if files[i].modTime.Equal(files[j].modTime) {
			return files[i].path < files[j].path
		}
		return files[i].modTime.Before(files[j].modTime)
	})
	for _, file := range files {
		if total <= maxDiskBytes {
			return
		}
		if file.path == activePath {
			continue
		}
		if err := os.Remove(file.path); err != nil {
			continue
		}
		total -= file.size
	}
}
