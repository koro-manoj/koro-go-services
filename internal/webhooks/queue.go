package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	key    string
}

func NewRedisQueue(client *redis.Client, key string) *RedisQueue {
	if key == "" {
		key = QueueKey
	}
	return &RedisQueue{client: client, key: key}
}

func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	if err := q.client.LPush(ctx, q.key, data).Err(); err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}
	return nil
}

func (q *RedisQueue) Dequeue(ctx context.Context) (Job, error) {
	result, err := q.client.BRPop(ctx, 0, q.key).Result()
	if err != nil {
		return Job{}, fmt.Errorf("dequeue job: %w", err)
	}
	if len(result) < 2 {
		return Job{}, fmt.Errorf("unexpected redis response")
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return Job{}, fmt.Errorf("unmarshal job: %w", err)
	}
	return job, nil
}
