package proxytoken

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/redis/go-redis/v9"
)

type Store interface {
	Put(ctx context.Context, tokenHash string, record domain.ProxyTokenRecord, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (domain.ProxyTokenRecord, bool)
}

type MemoryStore struct {
	mu    sync.Mutex
	items map[string]memoryItem
}

type memoryItem struct {
	record  domain.ProxyTokenRecord
	expires time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: map[string]memoryItem{}}
}

func (s *MemoryStore) Put(_ context.Context, tokenHash string, record domain.ProxyTokenRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[tokenHash] = memoryItem{record: record, expires: time.Now().UTC().Add(ttl)}
	return nil
}

func (s *MemoryStore) Get(_ context.Context, tokenHash string) (domain.ProxyTokenRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[tokenHash]
	if !ok || time.Now().UTC().After(item.expires) {
		delete(s.items, tokenHash)
		return domain.ProxyTokenRecord{}, false
	}
	return item.record, true
}

type RedisStore struct {
	client *redis.Client
	prefix string
}

func NewRedisStore(redisURL string) (*RedisStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisStore{client: redis.NewClient(options), prefix: "one_proxy:proxy_token:"}, nil
}

func (s *RedisStore) Put(ctx context.Context, tokenHash string, record domain.ProxyTokenRecord, ttl time.Duration) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.prefix+tokenHash, payload, ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, tokenHash string) (domain.ProxyTokenRecord, bool) {
	payload, err := s.client.Get(ctx, s.prefix+tokenHash).Bytes()
	if err != nil {
		return domain.ProxyTokenRecord{}, false
	}
	var record domain.ProxyTokenRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return domain.ProxyTokenRecord{}, false
	}
	return record, true
}
