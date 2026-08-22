package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type TaskRepo struct {
	db *sqlx.DB
}

func NewTaskRepo(db *sqlx.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, task *model.Task) error {
	now := timeutil.NowNaive()
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.Status == "" {
		task.Status = model.TaskCreated
	}
	if task.Stats == nil {
		task.Stats = model.JSONMap{}
	}
	q := `INSERT INTO tasks(name, rule_id, seed_urls, max_depth, concurrency, status, stats, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`
	err := r.db.QueryRowxContext(ctx, q,
		task.Name, task.RuleID, task.SeedURLs, task.MaxDepth, task.Concurrency,
		task.Status, task.Stats, task.CreatedAt, task.UpdatedAt,
	).Scan(&task.ID)
	return err
}

func (r *TaskRepo) Get(ctx context.Context, id int64) (*model.Task, error) {
	var out model.Task
	err := r.db.GetContext(ctx, &out, `SELECT * FROM tasks WHERE id=$1`, id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &out, err
}

func (r *TaskRepo) List(ctx context.Context, status string, limit, offset int) ([]model.Task, int64, error) {
	where := ""
	args := []any{}
	if status != "" {
		where = "WHERE status=$1"
		args = append(args, status)
	}
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM tasks "+where, args...); err != nil {
		return nil, 0, err
	}
	q := "SELECT * FROM tasks " + where + " ORDER BY id DESC"
	if where == "" {
		q += " LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	} else {
		q += " LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}
	var items []model.Task
	if err := r.db.SelectContext(ctx, &items, q, args...); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []model.Task{}
	}
	return items, total, nil
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id int64, from []string, to string) (*model.Task, error) {
	now := timeutil.NowNaive()
	q := `UPDATE tasks SET status=$1, updated_at=$2 WHERE id=$3`
	args := []any{to, now, id}
	if len(from) > 0 {
		q += " AND status IN ("
		for i, s := range from {
			if i > 0 {
				q += ","
			}
			q += fmt.Sprintf("$%d", len(args)+1)
			args = append(args, s)
		}
		q += ")"
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		task, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return task, fmt.Errorf("invalid status transition %s -> %s", task.Status, to)
	}
	return r.Get(ctx, id)
}

func (r *TaskRepo) AddStats(ctx context.Context, id int64, deltas map[string]int64) error {
	task, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if task.Stats == nil {
		task.Stats = model.JSONMap{}
	}
	for k, d := range deltas {
		task.Stats = task.Stats.Add(k, d)
	}
	now := timeutil.NowNaive()
	_, err = r.db.ExecContext(ctx, `UPDATE tasks SET stats=$1, updated_at=$2 WHERE id=$3`, task.Stats, now, id)
	return err
}

func (r *TaskRepo) SetStats(ctx context.Context, id int64, stats model.JSONMap) error {
	now := timeutil.NowNaive()
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET stats=$1, updated_at=$2 WHERE id=$3`, stats, now, id)
	return err
}

func (r *TaskRepo) ListByStatus(ctx context.Context, status string) ([]model.Task, error) {
	var items []model.Task
	if err := r.db.SelectContext(ctx, &items, `SELECT * FROM tasks WHERE status=$1 ORDER BY id`, status); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TaskRepo) Touch(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET updated_at=$1 WHERE id=$2`, timeutil.NowNaive(), id)
	return err
}

func EncodeStats(m model.JSONMap) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}
