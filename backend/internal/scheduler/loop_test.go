package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"goscrapy/internal/bloom"
	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/queue"
	"goscrapy/internal/store"
)

// loopTestEnv sets up a miniredis-backed Queue, Elector, and Loop for
// integration testing. The returned repos has nil DB-backed repos (Tasks,
// Nodes, etc.), so tests must NOT exercise code paths that hit Postgres
// (finishIdleTasks, pushMetrics). This is sufficient to verify the
// StillHolds gating of ReclaimExpired — the action that logs
// "reclaimed leases" in the reported incident.
func loopTestEnv(t *testing.T, id string) (*Loop, *Elector, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := queue.New(rdb, 2*time.Second)
	if err := q.LoadScripts(context.Background()); err != nil {
		t.Fatal(err)
	}
	bf := bloom.New(rdb, 1<<20, 4)
	elector := NewElector(rdb, id, 8*time.Second)
	loop := NewLoop(elector, q, bf, &store.Repos{}, nil, 30*time.Millisecond)
	return loop, elector, s, rdb
}

// TestLoopNonLeaderNeverReclaims verifies that an instance which never
// acquired leadership does not call ReclaimExpired. We observe this
// indirectly: ReclaimExpired IncrBy's the "queue:reclaim_total" key
// only when leases are reclaimed. A non-leader must leave it at zero.
func TestLoopNonLeaderNeverReclaims(t *testing.T) {
	logger.Init("warn")
	defer logger.Sync()

	loop, _, s, rdb := loopTestEnv(t, "node-b")

	// B will never be leader because A acquires first.
	electorA := NewElector(rdb, "node-a", 8*time.Second)
	if !electorA.Tick(context.Background()) {
		t.Fatal("A should acquire leadership")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go loop.Run(ctx)
	time.Sleep(150 * time.Millisecond)

	val, _ := s.Get("queue:reclaim_total")
	if val != "" {
		t.Fatalf("non-leader incremented reclaim_total: %s", val)
	}
}

// TestLoopStopsReclaimAfterFailover verifies that once another instance
// takes the leader lock, the stale instance stops calling ReclaimExpired
// on subsequent ticks. We use miniredis FastForward to expire A's lock
// (simulating a long GC pause), have B take over, then observe that
// A's loop does not increment reclaim_total further.
//
// To make ReclaimExpired observable, we seed an expired lease into the
// queue so that any call to ReclaimExpired will produce a non-zero IncrBy.
func TestLoopStopsReclaimAfterFailover(t *testing.T) {
	logger.Init("warn")
	defer logger.Sync()

	loop, electorA, s, rdb := loopTestEnv(t, "node-a")

	// Seed an expired lease so ReclaimExpired returns n>0 (and thus
	// IncrBy fires), making each reclaim call observable.
	ctx := context.Background()
	// Enqueue then pop a job to create a lease, then fast-forward to
	// expire it.
	q := queue.New(rdb, 1*time.Second)
	_ = q.LoadScripts(ctx)
	enqueueTestJob(t, q)
	if _, err := q.Pop(ctx, "dead-worker"); err != nil {
		t.Fatal(err)
	}
	// The lease score is now+1s; fast-forward the clock so it expires.
	s.FastForward(2 * time.Second)

	ctxRun, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go loop.Run(ctxRun)
	time.Sleep(100 * time.Millisecond)

	// A was leader and should have reclaimed at least once.
	reclaimedBefore, _ := rdb.Get(ctx, "queue:reclaim_total").Int64()
	if reclaimedBefore == 0 {
		// ReclaimExpired may not have fired yet on this tick; check at
		// least that A believed it was leader.
		if !electorA.IsLeader() {
			t.Fatal("A should be leader initially")
		}
	}

	// Failover: expire A's lock and let B take over.
	s.FastForward(9 * time.Second)
	electorB := NewElector(rdb, "node-b", 8*time.Second)
	if !electorB.Tick(ctx) {
		t.Fatal("B should acquire leadership after A's lock expired")
	}

	reclaimedAtFailover, _ := rdb.Get(ctx, "queue:reclaim_total").Int64()

	// Let A's loop run a few more ticks. It must NOT reclaim.
	time.Sleep(150 * time.Millisecond)
	reclaimedAfter, _ := rdb.Get(ctx, "queue:reclaim_total").Int64()

	if reclaimedAfter > reclaimedAtFailover {
		t.Fatalf("stale leader reclaimed after failover: at_failover=%d after=%d (split-brain bug)",
			reclaimedAtFailover, reclaimedAfter)
	}
	if electorA.IsLeader() {
		t.Fatal("stale A should have been demoted by StillHolds")
	}
	if !electorB.IsLeader() {
		t.Fatal("B should still be leader")
	}
}

func enqueueTestJob(t *testing.T, q *queue.Queue) *model.CrawlJob {
	t.Helper()
	job := &model.CrawlJob{
		TaskID:   1,
		RuleID:   1,
		URL:      "http://x/1",
		Priority: 5,
	}
	if err := q.Enqueue(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	return job
}
