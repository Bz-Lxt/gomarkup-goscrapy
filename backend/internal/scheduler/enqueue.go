package scheduler

import (
	"context"

	"goscrapy/internal/bloom"
	"goscrapy/internal/model"
	"goscrapy/internal/queue"
	"goscrapy/internal/store"
	"goscrapy/internal/urlx"
)

type Enqueuer struct {
	Queue *queue.Queue
	Bloom *bloom.Filter
	Tasks *store.TaskRepo
}

func (e *Enqueuer) Seeds(ctx context.Context, task *model.Task, rule *model.Rule) (int, error) {
	return EnqueueSeeds(ctx, e.Queue, e.Bloom, e.Tasks, task, rule)
}

func (e *Enqueuer) URL(ctx context.Context, task *model.Task, rule *model.Rule, raw string, depth, priority int) (bool, error) {
	return OfferURL(ctx, e.Queue, e.Bloom, e.Tasks, task, rule, raw, depth, priority)
}

// OfferURL 先入队再写入布隆，避免入队失败后 URL 被永久占位。
func OfferURL(ctx context.Context, q *queue.Queue, bf *bloom.Filter, tasks *store.TaskRepo, task *model.Task, rule *model.Rule, raw string, depth, priority int) (bool, error) {
	u := urlx.Canonical(raw)
	if u == "" {
		return false, nil
	}
	seen, err := bf.Test(ctx, task.ID, u)
	if err != nil {
		return false, err
	}
	if seen {
		if tasks != nil {
			_ = tasks.AddStats(ctx, task.ID, map[string]int64{"duplicated": 1})
		}
		return false, nil
	}
	job := &model.CrawlJob{
		TaskID:      task.ID,
		RuleID:      rule.ID,
		RuleVersion: rule.Version,
		URL:         u,
		Depth:       depth,
		Priority:    priority,
	}
	if err := q.Enqueue(ctx, job); err != nil {
		return false, err
	}
	if err := bf.Add(ctx, task.ID, u); err != nil {
		return false, err
	}
	if tasks != nil {
		_ = tasks.AddStats(ctx, task.ID, map[string]int64{"enqueued": 1})
	}
	return true, nil
}
