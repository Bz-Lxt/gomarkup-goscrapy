package store

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
)

func Open(ctx context.Context, dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

type Repos struct {
	DB      *sqlx.DB
	Rules   *RuleRepo
	Tasks   *TaskRepo
	Results *ResultRepo
	Nodes   NodeStore
	Audit   *AuditRepo
}

type NodeStore interface {
	Upsert(ctx context.Context, n *model.WorkerNode) error
	Touch(ctx context.Context, n *model.WorkerNode) error
	List(ctx context.Context) ([]model.WorkerNode, error)
	MarkStale(ctx context.Context, olderThan time.Duration) error
	Get(ctx context.Context, id string) (*model.WorkerNode, error)
}

func NewRepos(db *sqlx.DB) *Repos {
	return &Repos{
		DB:      db,
		Rules:   NewRuleRepo(db),
		Tasks:   NewTaskRepo(db),
		Results: NewResultRepo(db),
		Nodes:   NewNodeRepo(db),
		Audit:   NewAuditRepo(db),
	}
}
