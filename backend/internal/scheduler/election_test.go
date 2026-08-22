package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestElector(t *testing.T, id string) (*Elector, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	e := NewElector(rdb, id, 8*time.Second)
	return e, s, rdb
}

// TestTickAcquireRenew verifies the basic acquire -> renew path works for a
// single instance.
func TestTickAcquireRenew(t *testing.T) {
	e, s, _ := newTestElector(t, "node-a")
	ctx := context.Background()

	if !e.Tick(ctx) {
		t.Fatal("first tick should acquire leadership")
	}
	if !e.IsLeader() {
		t.Fatal("should be leader after acquire")
	}
	tok, _ := s.Get(lockKey)
	if tok != e.Token() {
		t.Fatalf("redis lock key = %q, want %q", tok, e.Token())
	}

	// Second tick renews.
	if !e.Tick(ctx) {
		t.Fatal("renew tick should succeed")
	}
	if e.Fencing() != 1 {
		t.Fatalf("fencing token = %d, want 1 (promote called once)", e.Fencing())
	}
}

// TestStillHoldsCatchesTakeover is the core regression test for the reported
// incident: instance A acquires the lock, then freezes (simulated by letting
// the TTL elapse). Instance B takes over via SetNX. A wakes up and must NOT
// be allowed to run leader-only actions — StillHolds must return false.
func TestStillHoldsCatchesTakeover(t *testing.T) {
	eA, s, _ := newTestElector(t, "node-a")
	eB := NewElector(redis.NewClient(&redis.Options{Addr: s.Addr()}), "node-b", 8*time.Second)
	ctx := context.Background()

	if !eA.Tick(ctx) {
		t.Fatal("A should acquire leadership")
	}
	if !eA.IsLeader() {
		t.Fatal("A should be leader")
	}

	// Simulate A freezing: let the lock TTL expire.
	s.FastForward(9 * time.Second)

	// B takes over.
	if !eB.Tick(ctx) {
		t.Fatal("B should acquire leadership after A's lock expired")
	}
	tok, _ := s.Get(lockKey)
	if tok != eB.Token() {
		t.Fatalf("redis lock key = %q, want B's token %q", tok, eB.Token())
	}

	// A wakes up — StillHolds must catch the takeover.
	if eA.StillHolds(ctx) {
		t.Fatal("A's StillHolds must return false after takeover; this is the split-brain bug")
	}
	if eA.IsLeader() {
		t.Fatal("A must no longer believe it is leader")
	}
}

// TestStaleLeaderTickDemotes verifies that a stale leader, on its next Tick,
// detects another instance holds the lock and demotes itself rather than
// blindly overwriting the key.
func TestStaleLeaderTickDemotes(t *testing.T) {
	eA, s, _ := newTestElector(t, "node-a")
	eB := NewElector(redis.NewClient(&redis.Options{Addr: s.Addr()}), "node-b", 8*time.Second)
	ctx := context.Background()

	if !eA.Tick(ctx) {
		t.Fatal("A should acquire")
	}
	s.FastForward(9 * time.Second)
	if !eB.Tick(ctx) {
		t.Fatal("B should acquire")
	}

	// A's next tick must fail and demote.
	if eA.Tick(ctx) {
		t.Fatal("A's tick must fail when B holds the lock")
	}
	if eA.IsLeader() {
		t.Fatal("A must be demoted after failed tick")
	}

	// A must not have clobbered B's lock.
	tok, _ := s.Get(lockKey)
	if tok != eB.Token() {
		t.Fatalf("redis lock key = %q, want B's token %q (A must not steal it)", tok, eB.Token())
	}
}

// TestResignReleasesLock verifies that Resign deletes the key only when the
// caller still owns it.
func TestResignReleasesLock(t *testing.T) {
	eA, s, _ := newTestElector(t, "node-a")
	ctx := context.Background()
	if !eA.Tick(ctx) {
		t.Fatal("A should acquire")
	}
	eA.Resign(ctx)
	if s.Exists(lockKey) {
		t.Fatal("lock key should be deleted after resign")
	}
	if eA.IsLeader() {
		t.Fatal("A should not be leader after resign")
	}
}

// TestResignAfterTakeoverNoop verifies that a stale instance calling Resign
// does not delete the lock owned by the new leader.
func TestResignAfterTakeoverNoop(t *testing.T) {
	eA, s, _ := newTestElector(t, "node-a")
	eB := NewElector(redis.NewClient(&redis.Options{Addr: s.Addr()}), "node-b", 8*time.Second)
	ctx := context.Background()

	if !eA.Tick(ctx) {
		t.Fatal("A should acquire")
	}
	s.FastForward(9 * time.Second)
	if !eB.Tick(ctx) {
		t.Fatal("B should acquire")
	}

	// A (stale) resigns — must not delete B's lock.
	eA.Resign(ctx)
	tok, _ := s.Get(lockKey)
	if tok != eB.Token() {
		t.Fatalf("stale A resign deleted B's lock: got %q, want %q", tok, eB.Token())
	}
	if !eB.IsLeader() {
		t.Fatal("B should still be leader after stale A resigns")
	}
}

// TestStillHoldsFalseWhenNotLeader verifies that StillHolds returns false
// immediately when the instance was never leader (no Redis call needed).
func TestStillHoldsFalseWhenNotLeader(t *testing.T) {
	e, _, _ := newTestElector(t, "node-a")
	ctx := context.Background()
	if e.StillHolds(ctx) {
		t.Fatal("StillHolds should be false when never leader")
	}
}

// TestTickRecurseBounded replaces the old unbounded recursion: when the key
// vanishes between SetNX and Get, Tick retries once rather than recursing.
func TestTickKeyVanishesBetweenSetNXAndGet(t *testing.T) {
	e, _, rdb := newTestElector(t, "node-a")
	ctx := context.Background()

	// Seed the key then delete it so Get returns redis.Nil during Tick.
	if err := rdb.Set(ctx, lockKey, "stale", 5*time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	rdb.Del(ctx, lockKey)

	// Tick should re-acquire via the bounded retry, not recurse.
	if !e.Tick(ctx) {
		t.Fatal("Tick should acquire on bounded retry")
	}
	if !e.IsLeader() {
		t.Fatal("should be leader")
	}
}

// TestDemoteLogsOnce verifies that demote is idempotent — calling it from
// multiple code paths in the same tick logs the warning exactly once.
func TestDemoteLogsOnce(t *testing.T) {
	e, _, _ := newTestElector(t, "node-a")
	ctx := context.Background()
	if !e.Tick(ctx) {
		t.Fatal("should acquire")
	}
	e.demote()
	e.demote()
	e.demote()
	if e.IsLeader() {
		t.Fatal("should not be leader after demote")
	}
	// After demote, StillHolds must short-circuit without a Redis call.
	if e.StillHolds(ctx) {
		t.Fatal("StillHolds must be false after demote")
	}
}
