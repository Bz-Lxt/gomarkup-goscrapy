package store

import (
	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

func RuleView(r *model.Rule) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":             r.ID,
		"name":           r.Name,
		"start_url":      r.StartURL,
		"item_selector":  r.ItemSelector,
		"link_selector":  r.LinkSelector,
		"fields":         r.Fields,
		"respect_robots": r.RespectRobots,
		"qps":            r.QPS,
		"version":        r.Version,
		"created_at":     timeutil.FormatNaive(r.CreatedAt),
		"updated_at":     timeutil.FormatNaive(r.UpdatedAt),
	}
}

func TaskView(t *model.Task) map[string]any {
	if t == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"rule_id":     t.RuleID,
		"seed_urls":   t.SeedURLs,
		"max_depth":   t.MaxDepth,
		"concurrency": t.Concurrency,
		"status":      t.Status,
		"stats":       t.Stats,
		"created_at":  timeutil.FormatNaive(t.CreatedAt),
		"updated_at":  timeutil.FormatNaive(t.UpdatedAt),
	}
}
