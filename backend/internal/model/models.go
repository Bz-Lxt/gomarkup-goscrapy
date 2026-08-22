package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goscrapy/internal/timeutil"
)

const (
	TaskCreated   = "created"
	TaskRunning   = "running"
	TaskPaused    = "paused"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"
	TaskCancelled = "cancelled"

	NodeOnline   = "online"
	NodeDegraded = "degraded"
	NodeOffline  = "offline"

	KindXPath = "xpath"
	KindCSS   = "css"
	KindRegex = "regex"
)

type FieldSpec struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Expr string `json:"expr"`
	Attr string `json:"attr,omitempty"`
}

func (f FieldSpec) Normalize() FieldSpec {
	f.Name = strings.TrimSpace(f.Name)
	f.Kind = strings.ToLower(strings.TrimSpace(f.Kind))
	f.Expr = strings.TrimSpace(f.Expr)
	f.Attr = strings.TrimSpace(f.Attr)
	if f.Attr == "" {
		f.Attr = "text"
	}
	return f
}

func (f FieldSpec) Valid() error {
	f = f.Normalize()
	if f.Name == "" {
		return fmt.Errorf("field name required")
	}
	if f.Expr == "" {
		return fmt.Errorf("field expr required")
	}
	switch f.Kind {
	case KindXPath, KindCSS, KindRegex:
	default:
		return fmt.Errorf("field kind must be xpath|css|regex")
	}
	return nil
}

type FieldList []FieldSpec

func (l FieldList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *FieldList) Scan(src any) error {
	if src == nil {
		*l = FieldList{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported FieldList type %T", src)
	}
	if len(raw) == 0 {
		*l = FieldList{}
		return nil
	}
	return json.Unmarshal(raw, l)
}

type StringList []string

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *StringList) Scan(src any) error {
	if src == nil {
		*l = StringList{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported StringList type %T", src)
	}
	if len(raw) == 0 {
		*l = StringList{}
		return nil
	}
	return json.Unmarshal(raw, l)
}

type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported JSONMap type %T", src)
	}
	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}
	return json.Unmarshal(raw, m)
}

func (m JSONMap) Int64(key string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		var n int64
		_, _ = fmt.Sscan(v, &n)
		return n
	default:
		return 0
	}
}

func (m JSONMap) Add(key string, delta int64) JSONMap {
	if m == nil {
		m = JSONMap{}
	}
	m[key] = float64(m.Int64(key) + delta)
	return m
}

type Rule struct {
	ID            int64     `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	StartURL      string    `db:"start_url" json:"start_url"`
	ItemSelector  string    `db:"item_selector" json:"item_selector"`
	LinkSelector  string    `db:"link_selector" json:"link_selector"`
	Fields        FieldList `db:"fields" json:"fields"`
	RespectRobots bool      `db:"respect_robots" json:"respect_robots"`
	QPS           float64   `db:"qps" json:"qps"`
	Version       int       `db:"version" json:"version"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type Task struct {
	ID          int64      `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	RuleID      int64      `db:"rule_id" json:"rule_id"`
	SeedURLs    StringList `db:"seed_urls" json:"seed_urls"`
	MaxDepth    int        `db:"max_depth" json:"max_depth"`
	Concurrency int        `db:"concurrency" json:"concurrency"`
	Status      string     `db:"status" json:"status"`
	Stats       JSONMap    `db:"stats" json:"stats"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

func (t Task) CanStart() bool {
	return t.Status == TaskCreated || t.Status == TaskPaused
}

func (t Task) CanPause() bool {
	return t.Status == TaskRunning
}

func (t Task) CanCancel() bool {
	return t.Status == TaskCreated || t.Status == TaskRunning || t.Status == TaskPaused
}

func (t Task) IsActive() bool {
	return t.Status == TaskRunning
}

type CrawlResult struct {
	ID        int64     `db:"id" json:"id"`
	TaskID    int64     `db:"task_id" json:"task_id"`
	URL       string    `db:"url" json:"url"`
	Payload   JSONMap   `db:"payload" json:"payload"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type WorkerNode struct {
	ID          string    `db:"id" json:"id"`
	Role        string    `db:"role" json:"role"`
	CPU         float64   `db:"cpu" json:"cpu"`
	MemoryMB    float64   `db:"memory_mb" json:"memory_mb"`
	PagesPerMin float64   `db:"pages_per_min" json:"pages_per_min"`
	FailRate    float64   `db:"fail_rate" json:"fail_rate"`
	Status      string    `db:"status" json:"status"`
	LastSeen    time.Time `db:"last_seen" json:"last_seen"`
}

func (n WorkerNode) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID          string  `json:"id"`
		Role        string  `json:"role"`
		CPU         float64 `json:"cpu"`
		MemoryMB    float64 `json:"memory_mb"`
		PagesPerMin float64 `json:"pages_per_min"`
		FailRate    float64 `json:"fail_rate"`
		Status      string  `json:"status"`
		LastSeen    string  `json:"last_seen"`
	}
	return json.Marshal(wire{
		ID:          n.ID,
		Role:        n.Role,
		CPU:         n.CPU,
		MemoryMB:    n.MemoryMB,
		PagesPerMin: n.PagesPerMin,
		FailRate:    n.FailRate,
		Status:      n.Status,
		LastSeen:    timeutil.FormatNaive(n.LastSeen),
	})
}

type AuditLog struct {
	ID        int64     `db:"id" json:"id"`
	Action    string    `db:"action" json:"action"`
	Detail    string    `db:"detail" json:"detail"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CrawlJob struct {
	ID          string `json:"id"`
	TaskID      int64  `json:"task_id"`
	RuleID      int64  `json:"rule_id"`
	RuleVersion int    `json:"rule_version"`
	URL         string `json:"url"`
	Depth       int    `json:"depth"`
	Priority    int    `json:"priority"`
	Attempt     int    `json:"attempt"`
	EnqueuedAt  int64  `json:"enqueued_at"`
}

const (
	ActionRobotsDisabled = "robots_disabled"
	ActionRuleCreated    = "rule_created"
	ActionRuleUpdated    = "rule_updated"
	ActionRuleDeleted    = "rule_deleted"
	ActionTaskCreated    = "task_created"
	ActionTaskStarted    = "task_started"
	ActionTaskPaused     = "task_paused"
	ActionTaskCancelled  = "task_cancelled"
)
