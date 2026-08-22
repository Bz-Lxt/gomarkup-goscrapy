package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"goscrapy/internal/model"
)

func testQueue(t *testing.T) (*Queue, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	q := New(rdb, 2*time.Second)
	if err := q.LoadScripts(context.Background()); err != nil {
		t.Fatal(err)
	}
	return q, s
}

func TestPopAck(t *testing.T) {
	q, _ := testQueue(t)
	ctx := context.Background()
	job := &model.CrawlJob{TaskID: 1, RuleID: 1, URL: "http://mock-target/list.html", Priority: 5}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatal(err)
	}
	got, err := q.Pop(ctx, "worker-1")
	if err != nil || got == nil {
		t.Fatalf("pop: %v %v", got, err)
	}
	if got.URL != job.URL {
		t.Fatalf("url=%s", got.URL)
	}
	st, err := q.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Ready != 0 || st.Leased != 1 {
		t.Fatalf("stats=%+v", st)
	}
	if err := q.Ack(ctx, got); err != nil {
		t.Fatal(err)
	}
	st, _ = q.Stats(ctx)
	if st.Leased != 0 {
		t.Fatalf("leased after ack: %d", st.Leased)
	}
}

func TestPriorityOrder(t *testing.T) {
	q, _ := testQueue(t)
	ctx := context.Background()
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 1, URL: "low", Priority: 1})
	time.Sleep(2 * time.Millisecond)
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 1, URL: "high", Priority: 9})
	first, err := q.Pop(ctx, "w")
	if err != nil || first == nil || first.URL != "high" {
		t.Fatalf("expected high first, got %+v err=%v", first, err)
	}
}

func TestReclaimExpired(t *testing.T) {
	q, mr := testQueue(t)
	ctx := context.Background()
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 7, URL: "http://x/1", Priority: 1})
	got, err := q.Pop(ctx, "dead-worker")
	if err != nil || got == nil {
		t.Fatalf("pop: %v %v", got, err)
	}
	// Lease expiry is a ZSET score (unix seconds), not a Redis TTL.
	// FastForward does not rewrite scores; move the member into the past.
	if _, err := mr.ZAdd(LeaseKey, 1, got.ID); err != nil {
		t.Fatal(err)
	}
	n, err := q.ReclaimExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("reclaimed=%d", n)
	}
	again, err := q.Pop(ctx, "worker-2")
	if err != nil || again == nil || again.URL != "http://x/1" {
		t.Fatalf("requeued job missing: %+v %v", again, err)
	}
}

func TestEmptyPop(t *testing.T) {
	q, _ := testQueue(t)
	got, err := q.Pop(context.Background(), "w")
	if err != nil || got != nil {
		t.Fatalf("empty pop should be nil,nil got %+v %v", got, err)
	}
}

func TestDropTask(t *testing.T) {
	q, _ := testQueue(t)
	ctx := context.Background()
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 3, URL: "a"})
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 3, URL: "b"})
	_ = q.Enqueue(ctx, &model.CrawlJob{TaskID: 4, URL: "c"})
	n, err := q.DropTask(ctx, 3)
	if err != nil || n != 2 {
		t.Fatalf("drop n=%d err=%v", n, err)
	}
	got, _ := q.Pop(ctx, "w")
	if got == nil || got.URL != "c" {
		t.Fatalf("expected leftover job c, got %+v", got)
	}
}
