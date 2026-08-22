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
	popSHA   string
	ackSHA   string
	recSHA   string
	dropSHA  string
}

func New(rdb *redis.Client, leaseTTL time.Duration) *Queue {
	if leaseTTL <= 0 || leaseTTL > 30*time.Second {
		leaseTTL = 30 * time.Second
	}
	return &Queue{rdb: rdb, leaseTTL: leaseTTL}
}

func (q *Queue) LoadScripts(ctx context.Context) error {
	var err error
	if q.popSHA, err = q.rdb.ScriptLoad(ctx, popLua).Result(); err != nil {
		return fmt.Errorf("load pop lua: %w", err)
	}
	if q.ackSHA, err = q.rdb.ScriptLoad(ctx, ackLua).Result(); err != nil {
		return fmt.Errorf("load ack lua: %w", err)
	}
	if q.recSHA, err = q.rdb.ScriptLoad(ctx, reclaimLua).Result(); err != nil {
		return fmt.Errorf("load reclaim lua: %w", err)
	}
	if q.dropSHA, err = q.rdb.ScriptLoad(ctx, dropTaskLua).Result(); err != nil {
		return fmt.Errorf("load drop lua: %w", err)
	}
	return nil
}

func (q *Queue) eval(ctx context.Context, sha, src string, keys []string, args ...any) *redis.Cmd {
	cmd := q.rdb.EvalSha(ctx, sha, keys, args...)
	if cmd.Err() != nil && isNOSCRIPT(cmd.Err()) {
		fallback := q.rdb.Eval(ctx, src, keys, args...)
		if fallback.Err() != nil {
			return fallback
		}
	}
	return cmd
}

func isNOSCRIPT(err error) bool {
	return err != nil && (err.Error() == "NOSCRIPT No matching script. Please use EVAL." ||
		contains(err.Error(), "NOSCRIPT"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > 0 && indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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
	res, err := q.eval(ctx, q.popSHA, popLua, []string{ReadyKey, PayloadKey, LeaseKey}, now, ttl, workerID).Result()
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
	_, err := q.eval(ctx, q.ackSHA, ackLua, []string{LeaseKey, PayloadKey, taskSetKey(job.TaskID)}, job.ID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("ack: %w", err)
	}
	return nil
}

func (q *Queue) ReclaimExpired(ctx context.Context) (int64, error) {
	now := timeutil.Unix(timeutil.Now())
	res, err := q.eval(ctx, q.recSHA, reclaimLua, []string{ReadyKey, LeaseKey, PayloadKey}, now).Result()
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
	res, err := q.eval(ctx, q.dropSHA, dropTaskLua, []string{ReadyKey, LeaseKey, PayloadKey, taskSetKey(taskID)}).Result()
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
