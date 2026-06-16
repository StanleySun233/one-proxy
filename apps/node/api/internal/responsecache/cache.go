package responsecache

import (
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Dir            string
	TTL            time.Duration
	MemoryMaxBytes int64
	DiskMaxBytes   int64
}

type Cache struct {
	dir            string
	ttl            time.Duration
	memoryMaxBytes int64
	diskMaxBytes   int64
	mu             sync.Mutex
	items          map[string]*list.Element
	order          *list.List
	memoryBytes    int64
	diskBytes      int64
}

type Entry struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	StoredAt   time.Time
	ExpiresAt  time.Time
}

type memoryEntry struct {
	key  string
	item Entry
	size int64
}

type diskMeta struct {
	StatusCode int         `json:"statusCode"`
	Header     http.Header `json:"header"`
	StoredAt   int64       `json:"storedAt"`
	ExpiresAt  int64       `json:"expiresAt"`
	BodySize   int64       `json:"bodySize"`
}

var ErrMiss = errors.New("cache_miss")

func New(cfg Config) (*Cache, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = time.Hour
	}
	if cfg.MemoryMaxBytes <= 0 {
		cfg.MemoryMaxBytes = 512 * 1024 * 1024
	}
	if cfg.DiskMaxBytes <= 0 {
		cfg.DiskMaxBytes = 2 * 1024 * 1024 * 1024
	}
	if cfg.Dir == "" {
		cfg.Dir = "runtime/cache/responses"
	}
	if err := os.RemoveAll(cfg.Dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{
		dir:            cfg.Dir,
		ttl:            cfg.TTL,
		memoryMaxBytes: cfg.MemoryMaxBytes,
		diskMaxBytes:   cfg.DiskMaxBytes,
		items:          make(map[string]*list.Element),
		order:          list.New(),
	}, nil
}

func Key(req *http.Request, body []byte) string {
	hasher := sha256.New()
	writePart(hasher, req.Method)
	if req.URL != nil {
		writePart(hasher, req.URL.String())
	}
	writePart(hasher, req.Host)
	if len(body) > 0 {
		sum := sha256.Sum256(body)
		writePart(hasher, hex.EncodeToString(sum[:]))
	}
	names := cacheKeyHeaderNames(req.Header)
	for _, name := range names {
		values := append([]string(nil), req.Header.Values(name)...)
		sort.Strings(values)
		writePart(hasher, strings.ToLower(name))
		for _, value := range values {
			writePart(hasher, value)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func CanStore(req *http.Request, statusCode int, header http.Header, bodySize int) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}
	if statusCode != http.StatusOK {
		return false
	}
	if bodySize < 0 {
		return false
	}
	if header.Get("Set-Cookie") != "" {
		return false
	}
	if header.Get("Vary") != "" {
		return false
	}
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	if strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "private") {
		return false
	}
	if strings.Contains(strings.ToLower(header.Get("Pragma")), "no-cache") {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(header.Get("Content-Type")))
	return !strings.HasPrefix(contentType, "text/event-stream")
}

func (c *Cache) Set(key string, entry Entry) error {
	if key == "" {
		return nil
	}
	bodySize := int64(len(entry.Body))
	if bodySize > c.diskMaxBytes {
		return nil
	}
	now := time.Now()
	entry.StoredAt = now
	entry.ExpiresAt = now.Add(c.ttl)
	entry.Header = entry.Header.Clone()
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.writeDiskLocked(key, entry); err != nil {
		return err
	}
	c.putMemoryLocked(key, entry)
	c.evictMemoryLocked()
	return c.evictDiskLocked()
}

func (c *Cache) Get(key string) (Entry, error) {
	if key == "" {
		return Entry{}, ErrMiss
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		item := element.Value.(*memoryEntry)
		if now.After(item.item.ExpiresAt) {
			c.removeMemoryLocked(element)
			_ = os.Remove(c.path(key))
			return Entry{}, ErrMiss
		}
		c.order.MoveToFront(element)
		return cloneEntry(item.item), nil
	}
	entry, err := c.readDiskLocked(key)
	if err != nil {
		return Entry{}, ErrMiss
	}
	if now.After(entry.ExpiresAt) {
		_ = os.Remove(c.path(key))
		return Entry{}, ErrMiss
	}
	c.putMemoryLocked(key, entry)
	c.evictMemoryLocked()
	return cloneEntry(entry), nil
}

func (c *Cache) writeDiskLocked(key string, entry Entry) error {
	path := c.path(key)
	if info, err := os.Stat(path); err == nil {
		c.diskBytes -= info.Size()
	}
	meta := diskMeta{
		StatusCode: entry.StatusCode,
		Header:     entry.Header.Clone(),
		StoredAt:   entry.StoredAt.UnixNano(),
		ExpiresAt:  entry.ExpiresAt.UnixNano(),
		BodySize:   int64(len(entry.Body)),
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	var headerLen [8]byte
	binary.BigEndian.PutUint64(headerLen[:], uint64(len(metaBytes)))
	if _, err = file.Write(headerLen[:]); err == nil {
		_, err = file.Write(metaBytes)
	}
	if err == nil {
		_, err = file.Write(entry.Body)
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if info, err := os.Stat(path); err == nil {
		c.diskBytes += info.Size()
	}
	return nil
}

func (c *Cache) readDiskLocked(key string) (Entry, error) {
	file, err := os.Open(c.path(key))
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	var headerLen [8]byte
	if _, err := io.ReadFull(file, headerLen[:]); err != nil {
		return Entry{}, err
	}
	metaBytes := make([]byte, binary.BigEndian.Uint64(headerLen[:]))
	if _, err := io.ReadFull(file, metaBytes); err != nil {
		return Entry{}, err
	}
	var meta diskMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return Entry{}, err
	}
	body, err := io.ReadAll(file)
	if err != nil {
		return Entry{}, err
	}
	if int64(len(body)) != meta.BodySize {
		return Entry{}, errors.New("cache_body_size_mismatch")
	}
	return Entry{
		StatusCode: meta.StatusCode,
		Header:     meta.Header.Clone(),
		Body:       body,
		StoredAt:   time.Unix(0, meta.StoredAt),
		ExpiresAt:  time.Unix(0, meta.ExpiresAt),
	}, nil
}

func (c *Cache) putMemoryLocked(key string, entry Entry) {
	size := entrySize(entry)
	if size > c.memoryMaxBytes {
		return
	}
	if element, ok := c.items[key]; ok {
		c.removeMemoryLocked(element)
	}
	item := &memoryEntry{key: key, item: cloneEntry(entry), size: size}
	element := c.order.PushFront(item)
	c.items[key] = element
	c.memoryBytes += size
}

func (c *Cache) evictMemoryLocked() {
	for c.memoryBytes > c.memoryMaxBytes {
		element := c.order.Back()
		if element == nil {
			return
		}
		c.removeMemoryLocked(element)
	}
}

func (c *Cache) removeMemoryLocked(element *list.Element) {
	item := element.Value.(*memoryEntry)
	delete(c.items, item.key)
	c.memoryBytes -= item.size
	c.order.Remove(element)
}

func (c *Cache) evictDiskLocked() error {
	if c.diskBytes <= c.diskMaxBytes {
		return nil
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	type diskItem struct {
		name    string
		modTime time.Time
		size    int64
	}
	items := make([]diskItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, diskItem{name: entry.Name(), modTime: info.ModTime(), size: info.Size()})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.Before(items[j].modTime)
	})
	for _, item := range items {
		if c.diskBytes <= c.diskMaxBytes {
			break
		}
		if err := os.Remove(filepath.Join(c.dir, item.name)); err == nil {
			c.diskBytes -= item.size
			if element, ok := c.items[strings.TrimSuffix(item.name, ".cache")]; ok {
				c.removeMemoryLocked(element)
			}
		}
	}
	return nil
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key+".cache")
}

func entrySize(entry Entry) int64 {
	size := int64(len(entry.Body))
	for key, values := range entry.Header {
		size += int64(len(key))
		for _, value := range values {
			size += int64(len(value))
		}
	}
	return size + 128
}

func cloneEntry(entry Entry) Entry {
	body := append([]byte(nil), entry.Body...)
	return Entry{
		StatusCode: entry.StatusCode,
		Header:     entry.Header.Clone(),
		Body:       body,
		StoredAt:   entry.StoredAt,
		ExpiresAt:  entry.ExpiresAt,
	}
}

func cacheKeyHeaderNames(header http.Header) []string {
	names := make([]string, 0, len(header))
	for name := range header {
		if skipCacheKeyHeader(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func skipCacheKeyHeader(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Cache-Control", "Connection", "Date", "If-Modified-Since", "If-None-Match", "Keep-Alive", "Pragma", "Proxy-Connection", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}

func writePart(writer io.Writer, value string) {
	_, _ = writer.Write([]byte(strconv.Itoa(len(value))))
	_, _ = writer.Write([]byte(":"))
	_, _ = writer.Write([]byte(value))
}
