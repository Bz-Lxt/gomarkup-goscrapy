package queue

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestPopNullPayloadIsNotReportedAsEmpty(t *testing.T) {
	q, _ := testQueue(t)
	ctx := context.Background()

	const jobID = "legacy-null-payload"
	if err := q.rdb.HSet(ctx, PayloadKey, jobID, "null").Err(); err != nil {
		t.Fatal(err)
	}
	if err := q.rdb.ZAdd(ctx, ReadyKey, redis.Z{Member: jobID}).Err(); err != nil {
		t.Fatal(err)
	}

	job, err := q.Pop(ctx, "worker-1")
	if err != nil {
		t.Fatalf("pop returned an error for a valid JSON payload: %v", err)
	}
	if job == nil {
		t.Fatal("pop reported an empty queue after leasing the ready task")
	}

	stats, err := q.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Ready != 0 || stats.Leased != 1 {
		t.Fatalf("unexpected queue state after pop: %+v", stats)
	}
}
