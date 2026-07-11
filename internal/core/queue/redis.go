package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

const redisQueueKey = "dataset-builder:jobs"

// RedisQueue implements Queue using Redis lists.
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue creates a Redis-backed queue.
func NewRedisQueue(addr, password string, db int) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisQueue{client: client}, nil
}

// Enqueue adds a job to the Redis queue.
func (q *RedisQueue) Enqueue(ctx context.Context, job domain.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, redisQueueKey, data).Err()
}

// Dequeue removes and returns a job from the queue (blocking).
func (q *RedisQueue) Dequeue(ctx context.Context) (domain.Job, error) {
	result, err := q.client.BRPop(ctx, 5*time.Second, redisQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return domain.Job{}, ErrEmpty
		}
		return domain.Job{}, err
	}
	if len(result) < 2 {
		return domain.Job{}, ErrEmpty
	}
	var job domain.Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

// Size returns the number of jobs in the queue.
func (q *RedisQueue) Size() int {
	ctx := context.Background()
	n, err := q.client.LLen(ctx, redisQueueKey).Result()
	if err != nil {
		return 0
	}
	return int(n)
}

// Close closes the Redis connection.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
