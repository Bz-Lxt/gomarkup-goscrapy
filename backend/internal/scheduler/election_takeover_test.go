package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestElectorStopsClaimingLeadershipAfterLeaseTakeover(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	oldLeader := NewElector(rdb, "scheduler-a", 3*time.Second)
	newLeader := NewElector(rdb, "scheduler-b", 3*time.Second)

	if !oldLeader.Tick(ctx) {
		t.Fatal("first scheduler did not acquire leadership")
	}
	mr.FastForward(4 * time.Second)
	if !newLeader.Tick(ctx) {
		t.Fatal("second scheduler did not acquire the expired lease")
	}
	if oldLeader.StillHolds(ctx) {
		t.Fatal("old scheduler still claims leadership after another scheduler acquired the lease")
	}
	if !newLeader.StillHolds(ctx) {
		t.Fatal("new scheduler should hold the lease")
	}
}
