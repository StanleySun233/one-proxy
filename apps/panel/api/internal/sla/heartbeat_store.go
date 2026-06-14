package sla

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type HeartbeatStore interface {
	Record(ctx context.Context, nodeID string, at time.Time) error
	Count(ctx context.Context, nodeID string, start time.Time, end time.Time) (int, error)
}

type MemoryHeartbeatStore struct {
	mu        sync.Mutex
	retention time.Duration
	interval  time.Duration
	items     map[string]map[int64]struct{}
}

func NewMemoryHeartbeatStore(retention time.Duration, interval time.Duration) *MemoryHeartbeatStore {
	return &MemoryHeartbeatStore{
		retention: retention,
		interval:  interval,
		items:     map[string]map[int64]struct{}{},
	}
}

func (s *MemoryHeartbeatStore) Record(_ context.Context, nodeID string, at time.Time) error {
	score := at.UTC().Truncate(s.interval).UnixMilli()
	cutoff := at.UTC().Add(-s.retention).UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.items[nodeID]
	if current == nil {
		current = map[int64]struct{}{}
	}
	for item := range current {
		if item < cutoff {
			delete(current, item)
		}
	}
	current[score] = struct{}{}
	s.items[nodeID] = current
	return nil
}

func (s *MemoryHeartbeatStore) Count(_ context.Context, nodeID string, start time.Time, end time.Time) (int, error) {
	from := start.UTC().UnixMilli()
	to := end.UTC().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for item := range s.items[nodeID] {
		if item >= from && item < to {
			count++
		}
	}
	return count, nil
}

type RedisHeartbeatStore struct {
	client    *redis.Client
	prefix    string
	retention time.Duration
	interval  time.Duration
}

func NewRedisHeartbeatStore(redisURL string, retention time.Duration, interval time.Duration) (*RedisHeartbeatStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisHeartbeatStore{
		client:    redis.NewClient(options),
		prefix:    "one_proxy:sla:heartbeat:",
		retention: retention,
		interval:  interval,
	}, nil
}

func (s *RedisHeartbeatStore) Record(ctx context.Context, nodeID string, at time.Time) error {
	at = at.UTC()
	key := s.key(nodeID)
	score := at.Truncate(s.interval).UnixMilli()
	cutoff := at.Add(-s.retention).UnixMilli()
	pipe := s.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(score), Member: strconv.FormatInt(score, 10)})
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoff, 10))
	pipe.Expire(ctx, key, s.retention)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisHeartbeatStore) Count(ctx context.Context, nodeID string, start time.Time, end time.Time) (int, error) {
	result, err := s.client.ZCount(
		ctx,
		s.key(nodeID),
		strconv.FormatInt(start.UTC().UnixMilli(), 10),
		strconv.FormatInt(end.UTC().Add(-time.Millisecond).UnixMilli(), 10),
	).Result()
	if err != nil {
		return 0, err
	}
	return int(result), nil
}

func (s *RedisHeartbeatStore) key(nodeID string) string {
	return s.prefix + nodeID
}
