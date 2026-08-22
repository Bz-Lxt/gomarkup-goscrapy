package model

import (
	"fmt"
	"strings"

	"goscrapy/internal/timeutil"
)

const (
	CodeOK            = 0
	CodeBadRequest    = 40001
	CodeUnauthorized  = 40101
	CodeBadCredential = 40102
	CodeNotFound      = 40401
	CodeConflict      = 40901
	CodeInternal      = 50001
	CodeUnavailable   = 50301
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginData struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	Username  string `json:"username"`
}

type HealthData struct {
	Status string `json:"status"`
	Role   string `json:"role"`
	Time   string `json:"time"`
}

type PageQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
	TaskID   int64  `form:"task_id"`
}

func (q *PageQuery) Normalize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Status = strings.TrimSpace(q.Status)
}

func (q PageQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

type PageData[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type RuleInput struct {
	Name          string    `json:"name"`
	StartURL      string    `json:"start_url"`
	ItemSelector  string    `json:"item_selector"`
	LinkSelector  string    `json:"link_selector"`
	Fields        FieldList `json:"fields"`
	RespectRobots *bool     `json:"respect_robots"`
	QPS           *float64  `json:"qps"`
}

func (in RuleInput) ValidateCreate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name required")
	}
	if strings.TrimSpace(in.StartURL) == "" {
		return fmt.Errorf("start_url required")
	}
	for i, f := range in.Fields {
		if err := f.Valid(); err != nil {
			return fmt.Errorf("fields[%d]: %w", i, err)
		}
	}
	if in.QPS != nil && *in.QPS <= 0 {
		return fmt.Errorf("qps must be > 0")
	}
	return nil
}

func (in RuleInput) Apply(dst *Rule) {
	if v := strings.TrimSpace(in.Name); v != "" {
		dst.Name = v
	}
	if v := strings.TrimSpace(in.StartURL); v != "" {
		dst.StartURL = v
	}
	if in.ItemSelector != "" || dst.ID == 0 {
		dst.ItemSelector = strings.TrimSpace(in.ItemSelector)
	}
	if in.LinkSelector != "" || dst.ID == 0 {
		dst.LinkSelector = strings.TrimSpace(in.LinkSelector)
	}
	if in.Fields != nil {
		norm := make(FieldList, 0, len(in.Fields))
		for _, f := range in.Fields {
			norm = append(norm, f.Normalize())
		}
		dst.Fields = norm
	}
	if in.RespectRobots != nil {
		dst.RespectRobots = *in.RespectRobots
	}
	if in.QPS != nil {
		dst.QPS = *in.QPS
	}
}

type PreviewRequest struct {
	HTML string `json:"html"`
	URL  string `json:"url"`
}

type TaskInput struct {
	Name        string     `json:"name"`
	RuleID      int64      `json:"rule_id"`
	SeedURLs    StringList `json:"seed_urls"`
	MaxDepth    int        `json:"max_depth"`
	Concurrency int        `json:"concurrency"`
}

func (in TaskInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name required")
	}
	if in.RuleID <= 0 {
		return fmt.Errorf("rule_id required")
	}
	if len(in.SeedURLs) == 0 {
		return fmt.Errorf("seed_urls required")
	}
	if in.MaxDepth < 0 {
		return fmt.Errorf("max_depth invalid")
	}
	if in.Concurrency < 0 {
		return fmt.Errorf("concurrency invalid")
	}
	return nil
}

func (in TaskInput) Normalize() TaskInput {
	in.Name = strings.TrimSpace(in.Name)
	if in.MaxDepth < 1 {
		in.MaxDepth = 1
	}
	if in.Concurrency < 1 {
		in.Concurrency = 2
	}
	seeds := make(StringList, 0, len(in.SeedURLs))
	seen := map[string]struct{}{}
	for _, u := range in.SeedURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		seeds = append(seeds, u)
	}
	in.SeedURLs = seeds
	return in
}

type SnapshotRequest struct {
	URL string `json:"url"`
}

type SnapshotData struct {
	SnapshotID string         `json:"snapshot_id"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	ImageURL   string         `json:"image_url"`
	Nodes      []SnapshotNode `json:"nodes"`
}

type SnapshotNode struct {
	NodeID int    `json:"node_id"`
	Tag    string `json:"tag"`
	Text   string `json:"text"`
	Box    Box    `json:"box"`
}

type Box struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type SelectorRequest struct {
	NodeID int `json:"node_id"`
}

type SelectorCandidate struct {
	Kind   string  `json:"kind"`
	Expr   string  `json:"expr"`
	Unique bool    `json:"unique"`
	Score  float64 `json:"score"`
}

type ListRule struct {
	ItemSelector  string `json:"item_selector"`
	FieldSelector string `json:"field_selector"`
	HitCount      int    `json:"hit_count"`
}

type SelectorData struct {
	Candidates []SelectorCandidate `json:"candidates"`
	ListRule   *ListRule           `json:"list_rule"`
}

type ProxyView struct {
	URL        string `json:"url"`
	Healthy    bool   `json:"healthy"`
	Hits       int64  `json:"hits"`
	Fails      int64  `json:"fails"`
	Evicted    bool   `json:"evicted"`
	LastError  string `json:"last_error,omitempty"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type QueueStats struct {
	Ready   int64 `json:"ready"`
	Leased  int64 `json:"leased"`
	Tasks   int64 `json:"tasks"`
	Reclaim int64 `json:"reclaim_total"`
}

type ClusterMetrics struct {
	Time        string       `json:"time"`
	Leader      bool         `json:"leader"`
	InstanceID  string       `json:"instance_id"`
	Nodes       []WorkerNode `json:"nodes"`
	Queue       QueueStats   `json:"queue"`
	PagesPerMin float64      `json:"pages_per_min"`
	FailRate    float64      `json:"fail_rate"`
}

func FormatTime(t interface{ IsZero() bool }) string {
	return ""
}

func NowString() string {
	return timeutil.Format(timeutil.Now())
}
