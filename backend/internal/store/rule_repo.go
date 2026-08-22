package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type RuleRepo struct {
	db *sqlx.DB
}

func NewRuleRepo(db *sqlx.DB) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) Create(ctx context.Context, rule *model.Rule) error {
	now := timeutil.NowNaive()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	if rule.Version == 0 {
		rule.Version = 1
	}
	if rule.QPS <= 0 {
		rule.QPS = 2
	}
	q := `INSERT INTO rules(name, start_url, item_selector, link_selector, fields, respect_robots, qps, version, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
	err := r.db.QueryRowxContext(ctx, q,
		rule.Name, rule.StartURL, rule.ItemSelector, rule.LinkSelector, rule.Fields,
		rule.RespectRobots, rule.QPS, rule.Version, rule.CreatedAt, rule.UpdatedAt,
	).Scan(&rule.ID)
	return mapError(err)
}

func (r *RuleRepo) Get(ctx context.Context, id int64) (*model.Rule, error) {
	var out model.Rule
	err := r.db.GetContext(ctx, &out, `SELECT * FROM rules WHERE id=$1`, id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &out, err
}

func (r *RuleRepo) GetByName(ctx context.Context, name string) (*model.Rule, error) {
	var out model.Rule
	err := r.db.GetContext(ctx, &out, `SELECT * FROM rules WHERE name=$1`, name)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &out, err
}

func (r *RuleRepo) List(ctx context.Context, keyword string, limit, offset int) ([]model.Rule, int64, error) {
	where := ""
	args := []any{}
	if keyword != "" {
		where = "WHERE name ILIKE $1 OR start_url ILIKE $1"
		args = append(args, "%"+keyword+"%")
	}
	var total int64
	countQ := "SELECT COUNT(*) FROM rules " + where
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}
	q := "SELECT * FROM rules " + where + " ORDER BY id DESC"
	if where == "" {
		q += " LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	} else {
		q += " LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}
	var items []model.Rule
	if err := r.db.SelectContext(ctx, &items, q, args...); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []model.Rule{}
	}
	return items, total, nil
}

func (r *RuleRepo) Update(ctx context.Context, rule *model.Rule) error {
	rule.UpdatedAt = timeutil.NowNaive()
	rule.Version++
	q := `UPDATE rules SET name=$1, start_url=$2, item_selector=$3, link_selector=$4, fields=$5,
		respect_robots=$6, qps=$7, version=$8, updated_at=$9 WHERE id=$10`
	res, err := r.db.ExecContext(ctx, q,
		rule.Name, rule.StartURL, rule.ItemSelector, rule.LinkSelector, rule.Fields,
		rule.RespectRobots, rule.QPS, rule.Version, rule.UpdatedAt, rule.ID,
	)
	if err != nil {
		return mapError(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RuleRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM rules WHERE id=$1`, id)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return fmt.Errorf("rule in use: %w", err)
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
