package store

import (
	"context"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type AuditRepo struct {
	db *sqlx.DB
}

func NewAuditRepo(db *sqlx.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Append(ctx context.Context, action, detail string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs(action, detail, created_at) VALUES($1,$2,$3)`,
		action, detail, timeutil.NowNaive())
	return err
}

func (r *AuditRepo) List(ctx context.Context, action string, limit int) ([]model.AuditLog, error) {
	if limit < 1 {
		limit = 50
	}
	var items []model.AuditLog
	var err error
	if action == "" {
		err = r.db.SelectContext(ctx, &items, `SELECT * FROM audit_logs ORDER BY id DESC LIMIT $1`, limit)
	} else {
		err = r.db.SelectContext(ctx, &items, `SELECT * FROM audit_logs WHERE action=$1 ORDER BY id DESC LIMIT $2`, action, limit)
	}
	if items == nil {
		items = []model.AuditLog{}
	}
	return items, err
}

func (r *AuditRepo) RobotsDisabled(ctx context.Context, ruleName string, ruleID int64) error {
	return r.Append(ctx, model.ActionRobotsDisabled,
		"rule="+ruleName+" id="+itoa(ruleID)+" respect_robots=false")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
