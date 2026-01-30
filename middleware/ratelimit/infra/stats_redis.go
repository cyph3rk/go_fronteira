package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"middleware-gateway/middleware/ratelimit/domain"

	"github.com/redis/go-redis/v9"
)

type RedisStatsStore struct {
	rdb *redis.Client

	prefix string
	// ttl aplica apenas em chaves de série temporal / por key.
	// total é cumulativo e não expira.
	ttl time.Duration

	bucket string // "minute" (padrão) ou "none"

	trackKeys bool
}

type RedisStatsOption func(*RedisStatsStore)

func WithStatsPrefix(prefix string) RedisStatsOption {
	return func(s *RedisStatsStore) {
		s.prefix = strings.Trim(prefix, ":")
	}
}

func WithStatsTTL(d time.Duration) RedisStatsOption {
	return func(s *RedisStatsStore) { s.ttl = d }
}

func WithStatsBucket(bucket string) RedisStatsOption {
	return func(s *RedisStatsStore) { s.bucket = strings.ToLower(strings.TrimSpace(bucket)) }
}

func WithStatsTrackKeys(track bool) RedisStatsOption {
	return func(s *RedisStatsStore) { s.trackKeys = track }
}

func NewRedisStatsStore(rdb *redis.Client, opts ...RedisStatsOption) *RedisStatsStore {
	s := &RedisStatsStore{
		rdb:    rdb,
		prefix: "ratelimit:stats",
		ttl:    24 * time.Hour,
		bucket: "minute",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *RedisStatsStore) Record(ctx context.Context, ev domain.StatsEvent) error {
	if s == nil || s.rdb == nil {
		return nil
	}

	at := ev.At
	if at.IsZero() {
		at = time.Now()
	}

	field := "denied"
	if ev.Allowed {
		field = "allowed"
	}

	totalKey := s.prefix + ":total"

	pipe := s.rdb.Pipeline()
	pipe.HIncrBy(ctx, totalKey, field, 1)

	if s.bucket == "minute" {
		bucketKey := fmt.Sprintf("%s:minute:%s", s.prefix, at.UTC().Format("200601021504"))
		pipe.HIncrBy(ctx, bucketKey, field, 1)
		if s.ttl > 0 {
			pipe.Expire(ctx, bucketKey, s.ttl)
		}
	}

	if ev.Method != "" || ev.Path != "" {
		routeKey := s.prefix + ":route"
		routeField := strings.TrimSpace(ev.Method) + " " + strings.TrimSpace(ev.Path)
		routeField = strings.TrimSpace(routeField)
		if routeField != "" {
			pipe.HIncrBy(ctx, routeKey, routeField+":"+field, 1)
		}
	}

	if s.trackKeys {
		k := strings.TrimSpace(string(ev.Key))
		if k != "" {
			keyKey := s.prefix + ":key:" + k
			pipe.HIncrBy(ctx, keyKey, field, 1)
			if s.ttl > 0 {
				pipe.Expire(ctx, keyKey, s.ttl)
			}
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}
