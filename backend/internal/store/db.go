package store

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
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
	Nodes   *NodeRepo
	Audit   *AuditRepo
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
