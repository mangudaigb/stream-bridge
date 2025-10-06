package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jibitesh/request-response-manager/configs"
	"github.com/jibitesh/request-response-manager/pkg/instance"
	"github.com/redis/go-redis/v9"
)

type SessionStore interface {
	Get(context context.Context, sessionId string) (*SessionInfo, error)
	Set(context context.Context, sessionId string, si *SessionInfo) error
	Refresh(context context.Context, sessionId string) error
	Delete(context context.Context, sessionId string) error
}

type RedisSessionStore struct {
	client   *redis.Client
	ttl      time.Duration
	instance *instance.Instance
}

var ErrNotFound = errors.New("session not found")

func NewRedisStore(cfg *configs.Config, instance *instance.Instance) (*RedisSessionStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, errors.New("redis: cannot connect to redis")
	}
	return &RedisSessionStore{
		client:   rdb,
		ttl:      cfg.Redis.Timeout,
		instance: instance,
	}, nil
}

func (r RedisSessionStore) redisKey(sessionId string) string {
	return fmt.Sprintf("session:%s", sessionId)
}

func (r RedisSessionStore) Set(ctx context.Context, sessionId string, si *SessionInfo) error {
	log.Printf("Setting session info %v for %s", si, sessionId)
	b, err := json.Marshal(si)
	if err != nil {
		log.Printf("Error marshalling session info: %v", err)
		return err
	}
	duration := r.ttl * time.Minute
	return r.client.Set(ctx, r.redisKey(sessionId), b, duration).Err()
}

func (r RedisSessionStore) Get(ctx context.Context, sessionId string) (*SessionInfo, error) {
	raws, err := r.client.Get(ctx, r.redisKey(sessionId)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var si SessionInfo
	err = json.Unmarshal([]byte(raws), &si)
	if err != nil {
		return nil, err
	}
	return &si, nil
}

func (r RedisSessionStore) Refresh(ctx context.Context, sessionId string) error {
	duration := r.ttl * time.Minute
	return r.client.Expire(ctx, r.redisKey(sessionId), duration).Err()
}

func (r RedisSessionStore) Delete(ctx context.Context, sessionId string) error {
	return r.client.Del(ctx, r.redisKey(sessionId)).Err()
}
