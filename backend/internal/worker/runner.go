package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"goscrapy/internal/bloom"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/logger"
	"goscrapy/internal/metrics"
	"goscrapy/internal/model"
	"goscrapy/internal/parser"
	"goscrapy/internal/queue"
	"goscrapy/internal/ratelimit"
	"goscrapy/internal/scheduler"
	"goscrapy/internal/store"
	"goscrapy/internal/urlx"
)

type Runner struct {
	id       string
	n        int
	queue    *queue.Queue
	bloom    *bloom.Filter
	repos    *store.Repos
	fetch    *fetcher.Client
	engine   *parser.Engine
	limit    *ratelimit.Limiter
	adapt    *ratelimit.Adaptive
	metrics  *metrics.Collector
	paused   map[int64]bool
	mu       sync.RWMutex
	shutdown bool
}

func New(id string, concurrency int, q *queue.Queue, bf *bloom.Filter, repos *store.Repos, fetch *fetcher.Client, limit *ratelimit.Limiter, adapt *ratelimit.Adaptive, col *metrics.Collector) *Runner {
	if concurrency < 1 {
		concurrency = 2
	}
	return &Runner{
		id:      id,
		n:       concurrency,
		queue:   q,
		bloom:   bf,
		repos:   repos,
		fetch:   fetch,
		engine:  parser.NewEngine(),
		limit:   limit,
		adapt:   adapt,
		metrics: col,
		paused:  map[int64]bool{},
	}
}

func (r *Runner) SetPaused(taskID int64, paused bool) {
	r.mu.Lock()
	r.paused[taskID] = paused
	r.mu.Unlock()
}

func (r *Runner) Shutdown() {
	r.mu.Lock()
	r.shutdown = true
	r.mu.Unlock()
}

func (r *Runner) stopped() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.shutdown
}

func (r *Runner) isPaused(taskID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.paused[taskID]
}

func (r *Runner) Run(ctx context.Context) {
	lg := logger.Named("worker").With(zap.String("id", r.id))
	var wg sync.WaitGroup
	for i := 0; i < r.n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.loop(ctx, lg)
		}()
	}
	wg.Wait()
}

func (r *Runner) loop(ctx context.Context, lg *zap.Logger) {
	idle := 200 * time.Millisecond
	for {
		if r.stopped() {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := r.queue.Pop(ctx, r.id)
		if err != nil {
			lg.Warn("pop failed", zap.Error(err))
			sleep(ctx, time.Second)
			continue
		}
		if job == nil {
			sleep(ctx, idle)
			continue
		}
		hold, err := r.handle(ctx, job, lg)
		if err != nil {
			lg.Warn("job failed", zap.Error(err), zap.String("url", job.URL), zap.Int64("task", job.TaskID))
			_ = r.repos.Tasks.AddStats(ctx, job.TaskID, map[string]int64{"failed": 1})
			if r.metrics != nil {
				r.metrics.ObservePage(true)
			}
		}
		if hold {
			continue
		}
		_ = r.queue.Ack(ctx, job)
	}
}

func (r *Runner) handle(ctx context.Context, job *model.CrawlJob, lg *zap.Logger) (bool, error) {
	task, err := r.repos.Tasks.Get(ctx, job.TaskID)
	if err != nil {
		return false, err
	}
	if task.Status == model.TaskPaused || r.isPaused(task.ID) || task.Status == model.TaskCreated {
		return true, nil
	}
	if !task.IsActive() {
		return false, nil
	}
	rule, err := r.repos.Rules.Get(ctx, job.RuleID)
	if err != nil {
		return false, err
	}
	domain := urlx.DomainKey(job.URL)
	qps := rule.QPS
	if r.adapt != nil {
		qps = r.adapt.QPS(domain)
		if qps <= 0 {
			qps = rule.QPS
		}
	}
	if err := r.limit.Wait(ctx, domain, qps); err != nil {
		return false, err
	}
	res, err := r.fetch.Fetch(ctx, job.URL, rule.RespectRobots)
	if err != nil {
		if r.adapt != nil {
			r.adapt.Observe(domain, 0, time.Second, rule.QPS)
		}
		return false, err
	}
	if res.RobotsSkip {
		_ = r.repos.Tasks.AddStats(ctx, task.ID, map[string]int64{"robots_skip": 1})
		return false, nil
	}
	if r.adapt != nil {
		r.adapt.Observe(domain, res.Status, res.Duration, rule.QPS)
	}
	fail := res.Status >= 400 || res.Blocked
	if r.metrics != nil {
		r.metrics.ObservePage(fail)
	}
	if fail {
		_ = r.repos.Tasks.AddStats(ctx, task.ID, map[string]int64{"failed": 1, "crawled": 1})
		return false, nil
	}
	extracted, err := r.engine.Extract(rule, res.FinalURL, res.Body)
	if err != nil {
		return false, err
	}
	saved := int64(0)
	for _, item := range extracted.Items {
		item["_url"] = res.FinalURL
		rec := &model.CrawlResult{TaskID: task.ID, URL: res.FinalURL, Payload: item}
		if err := r.repos.Results.Insert(ctx, rec); err != nil {
			lg.Warn("insert result", zap.Error(err))
			continue
		}
		saved++
	}
	_ = r.repos.Tasks.AddStats(ctx, task.ID, map[string]int64{"crawled": 1, "results": saved})

	if job.Depth >= task.MaxDepth {
		return false, nil
	}
	for _, link := range extracted.Links {
		if _, err := scheduler.OfferURL(ctx, r.queue, r.bloom, r.repos.Tasks, task, rule, link, job.Depth+1, 5); err != nil {
			lg.Warn("enqueue child", zap.Error(err))
		}
	}
	return false, nil
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	select {
	case <-ctx.Done():
		t.Stop()
	case <-t.C:
	}
}
