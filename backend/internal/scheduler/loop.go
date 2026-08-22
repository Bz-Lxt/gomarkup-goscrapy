package scheduler

import (
	"context"
	"time"

	"go.uber.org/zap"

	"goscrapy/internal/bloom"
	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/queue"
	"goscrapy/internal/store"
	"goscrapy/internal/urlx"
	"goscrapy/internal/ws"
)

type Loop struct {
	elector  *Elector
	queue    *queue.Queue
	bloom    *bloom.Filter
	repos    *store.Repos
	hub      *ws.Hub
	interval time.Duration
	stale    time.Duration
}

func NewLoop(e *Elector, q *queue.Queue, b *bloom.Filter, repos *store.Repos, hub *ws.Hub, interval time.Duration) *Loop {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Loop{elector: e, queue: q, bloom: b, repos: repos, hub: hub, interval: interval, stale: 20 * time.Second}
}

func (l *Loop) Run(ctx context.Context) {
	lg := logger.Named("scheduler")
	t := time.NewTicker(l.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			leader := l.elector.Tick(ctx)
			if !leader {
				continue
			}
			if !l.elector.StillHolds(ctx) {
				continue
			}
			if n, err := l.queue.ReclaimExpired(ctx); err != nil {
				lg.Warn("reclaim failed", zap.Error(err))
			} else if n > 0 {
				lg.Info("reclaimed leases", zap.Int64("n", n))
			}
			l.finishIdleTasks(ctx)
			if err := l.repos.Nodes.MarkStale(ctx, l.stale); err != nil {
				lg.Debug("mark stale nodes", zap.Error(err))
			}
			l.pushMetrics(ctx)
		}
	}
}

func (l *Loop) finishIdleTasks(ctx context.Context) {
	tasks, err := l.repos.Tasks.ListByStatus(ctx, model.TaskRunning)
	if err != nil {
		return
	}
	for _, t := range tasks {
		pending, err := l.queue.TaskPending(ctx, t.ID)
		if err != nil {
			continue
		}
		if pending > 0 {
			continue
		}
		progress := t.Stats.Int64("crawled") + t.Stats.Int64("failed") + t.Stats.Int64("robots_skip")
		if progress == 0 {
			continue
		}
		if _, err := l.repos.Tasks.UpdateStatus(ctx, t.ID, []string{model.TaskRunning}, model.TaskSucceeded); err != nil {
			logger.Named("scheduler").Debug("complete task", zap.Error(err), zap.Int64("task", t.ID))
		} else {
			logger.Named("scheduler").Info("task succeeded", zap.Int64("id", t.ID))
		}
	}
}

func (l *Loop) pushMetrics(ctx context.Context) {
	if l.hub == nil {
		return
	}
	nodes, err := l.repos.Nodes.List(ctx)
	if err != nil {
		return
	}
	l.hub.BroadcastMetrics(nodes)
}

func EnqueueSeeds(ctx context.Context, q *queue.Queue, bf *bloom.Filter, tasks *store.TaskRepo, task *model.Task, rule *model.Rule) (int, error) {
	n := 0
	for i, raw := range task.SeedURLs {
		u := urlx.Canonical(raw)
		if u == "" {
			continue
		}
		ok, err := OfferURL(ctx, q, bf, tasks, task, rule, u, 0, 10-i)
		if err != nil {
			return n, err
		}
		if ok {
			n++
		}
	}
	return n, nil
}
