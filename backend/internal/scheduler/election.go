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

// Tick attempts to acquire or renew the leader lock. It returns true only
// when Redis confirms this instance's token still owns the key, so a lock
// taken by another instance is never mistaken for our own.
func (e *Elector) Tick(ctx context.Context) bool {
	val := e.token
	ok, err := e.rdb.SetNX(ctx, lockKey, val, e.ttl).Result()
	if err != nil {
		logger.Named("election").Warn("setnx failed", zap.Error(err))
		e.demote()
		return false
	}
	if ok {
		e.promote()
		return true
	}
	// Key exists — verify it still belongs to us.
	cur, err := e.rdb.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		// Key expired between SetNX and Get; retry acquisition once
		// instead of recursing indefinitely.
		ok2, err2 := e.rdb.SetNX(ctx, lockKey, val, e.ttl).Result()
		if err2 != nil || !ok2 {
			e.demote()
			return false
		}
		e.promote()
		return true
	}
	if err != nil {
		e.demote()
		return false
	}
	if cur == val {
		if err := e.rdb.Expire(ctx, lockKey, e.ttl).Err(); err != nil {
			e.demote()
			return false
		}
		e.leader.Store(true)
		return true
	}
	// Another instance holds the lock.
	e.demote()
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

// demote clears leadership state. The warning is logged exactly once (via
// CompareAndSwap) so the transition is visible even when demote is called
// from multiple code paths within the same tick.
func (e *Elector) demote() {
	if e.leader.CompareAndSwap(true, false) {
		logger.Named("election").Warn("lost leadership",
			zap.String("token", e.token),
			zap.String("at", timeutil.Format(timeutil.Now())))
	}
}

func (e *Elector) Resign(ctx context.Context) {
	script := redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`)
	_ = script.Run(ctx, e.rdb, []string{lockKey}, e.token).Err()
	e.demote()
}

// StillHolds re-verifies against Redis that this instance still owns the
// leader lock. Unlike reading a local flag, this catches lease loss that
// occurred between ticks — e.g. a long GC pause after Tick returned true,
// during which the TTL expired and another instance took over. Every
// leader-only action should call this immediately before touching shared
// state.
func (e *Elector) StillHolds(ctx context.Context) bool {
	if !e.leader.Load() {
		return false
	}
	cur, err := e.rdb.Get(ctx, lockKey).Result()
	if err != nil || cur != e.token {
		e.demote()
		return false
	}
	return true
}

func LockValue(id, nonce string) string {
	return fmt.Sprintf("%s:%s", id, nonce)
}
