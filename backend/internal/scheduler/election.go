package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"goscrapy/internal/logger"
	"goscrapy/internal/timeutil"
)

const lockKey = "goscrapy:election:master"

type Elector struct {
	rdb     *redis.Client
	ttl     time.Duration
	id      string
	token   string
	leader  atomic.Bool
	mu      sync.Mutex
	fencing int64
}

func NewElector(rdb *redis.Client, instanceID string, ttl time.Duration) *Elector {
	if ttl < 3*time.Second {
		ttl = 8 * time.Second
	}
	tok := make([]byte, 8)
	_, _ = rand.Read(tok)
	return &Elector{
		rdb:   rdb,
		ttl:   ttl,
		id:    instanceID,
		token: instanceID + ":" + hex.EncodeToString(tok),
	}
}

func (e *Elector) IsLeader() bool { return e.leader.Load() }

func (e *Elector) Token() string { return e.token }

func (e *Elector) Fencing() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.fencing
}

func (e *Elector) Tick(ctx context.Context) bool {
	val := e.token
	ok, err := e.rdb.SetNX(ctx, lockKey, val, e.ttl).Result()
	if err != nil {
		logger.Named("election").Warn("setnx failed", zap.Error(err))
		e.leader.Store(false)
		return false
	}
	if ok {
		e.promote()
		return true
	}
	cur, err := e.rdb.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return e.Tick(ctx)
	}
	if err != nil {
		e.leader.Store(false)
		return false
	}
	if cur == val {
		if err := e.rdb.Expire(ctx, lockKey, e.ttl).Err(); err != nil {
			e.leader.Store(false)
			return false
		}
		e.leader.Store(true)
		return true
	}
	e.leader.Store(false)
	return false
}

func (e *Elector) promote() {
	e.mu.Lock()
	e.fencing++
	e.mu.Unlock()
	e.leader.Store(true)
	logger.Named("election").Info("became leader",
		zap.String("token", e.token),
		zap.String("at", timeutil.Format(timeutil.Now())))
}

func (e *Elector) Resign(ctx context.Context) {
	script := redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`)
	_ = script.Run(ctx, e.rdb, []string{lockKey}, e.token).Err()
	e.leader.Store(false)
}

func (e *Elector) StillHolds(ctx context.Context) bool {
	cur, err := e.rdb.Get(ctx, lockKey).Result()
	if err != nil {
		return false
	}
	return cur == e.token
}

func LockValue(id, nonce string) string {
	return fmt.Sprintf("%s:%s", id, nonce)
}
