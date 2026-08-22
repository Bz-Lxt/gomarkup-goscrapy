package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type Queue struct {
	rdb      *redis.Client
	leaseTTL time.Duration
	pop      *redis.Script
	ack      *redis.Script
	reclaim  *redis.Script
	drop     *redis.Script
}

func New(rdb *redis.Client, leaseTTL time.Duration) *Queue {
	if leaseTTL <= 0 || leaseTTL > 30*time.Second {
		leaseTTL = 30 * time.Second
	}
	return &Queue{
		rdb:      rdb,
		leaseTTL: leaseTTL,
		pop:      redis.NewScript(popLua),
		ack:      redis.NewScript(ackLua),
		reclaim:  redis.NewScript(reclaimLua),
		drop:     redis.NewScript(dropTaskLua),
	}
}

func (q *Queue) LoadScripts(ctx context.Context) error {
	scripts := []*redis.Script{q.pop, q.ack, q.reclaim, q.drop}
	for _, s := range scripts {
		if err := s.Load(ctx, q.rdb).Err(); err != nil {
			return fmt.Errorf("load lua script: %w", err)
		}
	}
	return nil
}

// runScript executes a Lua script using the optimistic EVALSHA + EVAL-fallback
// pattern from go-redis. Script.Run first tries EVALSHA; on NOSCRIPT it
// transparently retries with EVAL, returning the Cmd whose result (value or
// error) reflects the actual outcome — never the stale NOSCRIPT error.
func (q *Queue) runScript(ctx context.Context, s *redis.Script, keys []string, args ...any) *redis.Cmd {
	return s.Run(ctx, q.rdb, keys, args...)
}

func (q *Queue) Enqueue(ctx context.Context, job *model.CrawlJob) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.EnqueuedAt == 0 {
		job.EnqueuedAt = timeutil.Unix(timeutil.Now())
	}
	raw, err := encodeJob(job)
	if err != nil {
		return err
	}
	score := float64(-job.Priority)*1e13 + float64(job.EnqueuedAt) + float64(job.Attempt)*1e-3
	pipe := q.rdb.TxPipeline()
	pipe.HSet(ctx, PayloadKey, job.ID, raw)
	pipe.ZAdd(ctx, ReadyKey, redis.Z{Score: score, Member: job.ID})
	pipe.SAdd(ctx, taskSetKey(job.TaskID), job.ID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("enqueue: %w", err)
	}
	return nil
}

func (q *Queue) Pop(ctx context.Context, workerID string) (*model.CrawlJob, error) {
	now := timeutil.Unix(timeutil.Now())
	ttl := int(q.leaseTTL.Seconds())
	if ttl < 1 {
		ttl = 30
	}
	res, err := q.runScript(ctx, q.pop, []string{ReadyKey, PayloadKey, LeaseKey}, now, ttl, workerID).Result()
	if err == redis.Nil || res == nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pop: %w", err)
	}
	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return nil, nil
	}
	raw, _ := arr[1].(string)
	if raw == "" {
		if b, ok := arr[1].([]byte); ok {
			raw = string(b)
		}
	}
	job, err := decodeJob(raw)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (q *Queue) Ack(ctx context.Context, job *model.CrawlJob) error {
	if job == nil {
		return nil
	}
	_, err := q.runScript(ctx, q.ack, []string{LeaseKey, PayloadKey, taskSetKey(job.TaskID)}, job.ID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("ack: %w", err)
	}
	return nil
}

func (q *Queue) ReclaimExpired(ctx context.Context) (int64, error) {
	now := timeutil.Unix(timeutil.Now())
	res, err := q.runScript(ctx, q.reclaim, []string{ReadyKey, LeaseKey, PayloadKey}, now).Result()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("reclaim: %w", err)
	}
	n := toInt64(res)
	if n > 0 {
		_ = q.rdb.IncrBy(ctx, ReclaimKey, n).Err()
	}
	return n, nil
}

func (q *Queue) DropTask(ctx context.Context, taskID int64) (int64, error) {
	res, err := q.runScript(ctx, q.drop, []string{ReadyKey, LeaseKey, PayloadKey, taskSetKey(taskID)}).Result()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("drop task: %w", err)
	}
	return toInt64(res), nil
}

func (q *Queue) Stats(ctx context.Context) (model.QueueStats, error) {
	ready, err := q.rdb.ZCard(ctx, ReadyKey).Result()
	if err != nil {
		return model.QueueStats{}, err
	}
	leased, err := q.rdb.ZCard(ctx, LeaseKey).Result()
	if err != nil {
		return model.QueueStats{}, err
	}
	reclaim, _ := q.rdb.Get(ctx, ReclaimKey).Int64()
	return model.QueueStats{Ready: ready, Leased: leased, Reclaim: reclaim}, nil
}

func (q *Queue) ReadyCount(ctx context.Context) (int64, error) {
	return q.rdb.ZCard(ctx, ReadyKey).Result()
}

func (q *Queue) LeasedCount(ctx context.Context) (int64, error) {
	return q.rdb.ZCard(ctx, LeaseKey).Result()
}

func (q *Queue) TaskPending(ctx context.Context, taskID int64) (int64, error) {
	return q.rdb.SCard(ctx, taskSetKey(taskID)).Result()
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
