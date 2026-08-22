package store

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type NodeRepo struct {
	db *sqlx.DB
}

func NewNodeRepo(db *sqlx.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

func (r *NodeRepo) Upsert(ctx context.Context, n *model.WorkerNode) error {
	return r.upsert(ctx, n, true)
}

func (r *NodeRepo) Touch(ctx context.Context, n *model.WorkerNode) error {
	return r.upsert(ctx, n, false)
}

func (r *NodeRepo) upsert(ctx context.Context, n *model.WorkerNode, withMetrics bool) error {
	n.LastSeen = timeutil.NowNaive()
	if n.Role == "" {
		n.Role = "worker"
	}
	if n.Status == "" {
		n.Status = model.NodeOnline
	}
	if withMetrics {
		q := `INSERT INTO worker_nodes(id, role, cpu, memory_mb, pages_per_min, fail_rate, status, last_seen)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (id) DO UPDATE SET
				role=EXCLUDED.role,
				cpu=EXCLUDED.cpu,
				memory_mb=EXCLUDED.memory_mb,
				pages_per_min=EXCLUDED.pages_per_min,
				fail_rate=EXCLUDED.fail_rate,
				status=EXCLUDED.status,
				last_seen=EXCLUDED.last_seen`
		_, err := r.db.ExecContext(ctx, q, n.ID, n.Role, n.CPU, n.MemoryMB, n.PagesPerMin, n.FailRate, n.Status, n.LastSeen)
		return err
	}
	q := `INSERT INTO worker_nodes(id, role, cpu, memory_mb, pages_per_min, fail_rate, status, last_seen)
		VALUES($1,$2,0,0,0,0,$3,$4)
		ON CONFLICT (id) DO UPDATE SET
			role=EXCLUDED.role,
			status=EXCLUDED.status,
			last_seen=EXCLUDED.last_seen`
	_, err := r.db.ExecContext(ctx, q, n.ID, n.Role, n.Status, n.LastSeen)
	return err
}

func (r *NodeRepo) List(ctx context.Context) ([]model.WorkerNode, error) {
	var items []model.WorkerNode
	err := r.db.SelectContext(ctx, &items, `SELECT * FROM worker_nodes ORDER BY id`)
	if items == nil {
		items = []model.WorkerNode{}
	}
	return items, err
}

func (r *NodeRepo) MarkStale(ctx context.Context, olderThan time.Duration) error {
	cutoff := timeutil.Naive(timeutil.Now().Add(-olderThan))
	_, err := r.db.ExecContext(ctx,
		`UPDATE worker_nodes SET status=$1 WHERE last_seen < $2 AND status <> $1`,
		model.NodeOffline, cutoff)
	return err
}

func (r *NodeRepo) Get(ctx context.Context, id string) (*model.WorkerNode, error) {
	var n model.WorkerNode
	err := r.db.GetContext(ctx, &n, `SELECT * FROM worker_nodes WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
