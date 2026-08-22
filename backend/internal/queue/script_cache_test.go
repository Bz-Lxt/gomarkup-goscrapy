package queue

import (
	"context"
	"testing"

	"goscrapy/internal/model"
)

func TestPopRecoversAfterScriptCacheFlush(t *testing.T) {
	q, _ := testQueue(t)
	ctx := context.Background()
	job := &model.CrawlJob{TaskID: 42, URL: "https://example.test/items/1"}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := q.rdb.ScriptFlush(ctx).Err(); err != nil {
		t.Fatal(err)
	}

	got, err := q.Pop(ctx, "worker-after-maintenance")
	if err != nil {
		t.Fatalf("pop after script cache flush: %v", err)
	}
	if got == nil || got.ID != job.ID {
		t.Fatalf("expected queued job %q, got %+v", job.ID, got)
	}

	stats, err := q.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Ready != 0 || stats.Leased != 1 {
		t.Fatalf("unexpected queue state after pop: %+v", stats)
	}
}
