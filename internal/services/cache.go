package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
)

// CacheService manages the ValKey (Redis-compatible) cache connection.
type CacheService struct {
	cfg    config.CacheConfig
	client *redis.Client
	log    *logging.Logger
}

// NewCacheService creates a new CacheService.
func NewCacheService(cfg config.CacheConfig, log *logging.Logger) *CacheService {
	return &CacheService{cfg: cfg, log: log}
}

func (s *CacheService) Name() string { return "cache" }

func (s *CacheService) Connect(ctx context.Context) error {
	s.log.Debug("Dialing cache", "addr", s.cfg.Address())
	s.client = redis.NewClient(&redis.Options{
		Addr:         s.cfg.Address(),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	if err := s.client.Ping(ctx).Err(); err != nil {
		return i18n.Err(i18n.ErrCachePing, err)
	}
	return nil
}

// SelfTest sets the "init" key with the current date/time and reads it back.
func (s *CacheService) SelfTest(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)

	s.log.Debug(i18n.Get(i18n.MsgCacheSelfTestInit), "value", now)

	if err := s.client.Set(ctx, "init", now, 0).Err(); err != nil {
		return i18n.Err(i18n.ErrCacheSetInit, err)
	}

	val, err := s.client.Get(ctx, "init").Result()
	if err != nil {
		return i18n.Err(i18n.ErrCacheGetInit, err)
	}
	if val != now {
		return fmt.Errorf("%s: got %q, want %q", i18n.Get(i18n.ErrCacheInitMismatch), val, now)
	}

	s.log.Info(i18n.Get(i18n.MsgCacheSelfTestPassed))
	return nil
}

func (s *CacheService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *CacheService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// Client returns the underlying Redis client for reuse by other packages.
func (s *CacheService) Client() *redis.Client {
	return s.client
}
