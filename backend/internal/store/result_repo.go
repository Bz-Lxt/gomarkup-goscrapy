package store

import (
	"context"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type ResultRepo struct {
	db *sqlx.DB
}

func NewResultRepo(db *sqlx.DB) *ResultRepo {
	return &ResultRepo{db: db}
}

func (r *ResultRepo) Insert(ctx context.Context, rec *model.CrawlResult) error {
	rec.CreatedAt = timeutil.NowNaive()
	if rec.Payload == nil {
		rec.Payload = model.JSONMap{}
	}
	q := `INSERT INTO crawl_results(task_id, url, payload, created_at) VALUES($1,$2,$3,$4) RETURNING id`
	return r.db.QueryRowxContext(ctx, q, rec.TaskID, rec.URL, rec.Payload, rec.CreatedAt).Scan(&rec.ID)
}

func (r *ResultRepo) ListByTask(ctx context.Context, taskID int64, limit, offset int) ([]model.CrawlResult, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM crawl_results WHERE task_id=$1`, taskID); err != nil {
		return nil, 0, err
	}
	var items []model.CrawlResult
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM crawl_results WHERE task_id=$1 ORDER BY id DESC LIMIT $2 OFFSET $3`,
		taskID, limit, offset)
	if items == nil {
		items = []model.CrawlResult{}
	}
	return items, total, err
}

func (r *ResultRepo) List(ctx context.Context, taskID int64, limit, offset int) ([]model.CrawlResult, int64, error) {
	if taskID > 0 {
		return r.ListByTask(ctx, taskID, limit, offset)
	}
	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM crawl_results`); err != nil {
		return nil, 0, err
	}
	var items []model.CrawlResult
	err := r.db.SelectContext(ctx, &items, `SELECT * FROM crawl_results ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if items == nil {
		items = []model.CrawlResult{}
	}
	return items, total, err
}

func (r *ResultRepo) CountByTask(ctx context.Context, taskID int64) (int64, error) {
	var n int64
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM crawl_results WHERE task_id=$1`, taskID)
	return n, err
}
