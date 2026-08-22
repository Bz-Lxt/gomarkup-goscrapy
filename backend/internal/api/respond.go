package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
	"goscrapy/internal/store"
	"goscrapy/internal/timeutil"
)

func OK(c *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	c.JSON(http.StatusOK, model.Envelope{Code: model.CodeOK, Message: "ok", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, model.Envelope{Code: model.CodeOK, Message: "ok", Data: data})
}

func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, model.Envelope{Code: model.CodeOK, Message: "ok", Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, model.Envelope{Code: code, Message: msg, Data: map[string]any{}})
}

func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, model.CodeBadRequest, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = "未登录或令牌失效"
	}
	Fail(c, http.StatusUnauthorized, model.CodeUnauthorized, msg)
}

func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "资源不存在"
	}
	Fail(c, http.StatusNotFound, model.CodeNotFound, msg)
}

func Conflict(c *gin.Context, msg string) {
	if msg == "" {
		msg = "资源冲突"
	}
	Fail(c, http.StatusConflict, model.CodeConflict, msg)
}

func Internal(c *gin.Context, msg string) {
	if msg == "" {
		msg = "内部错误"
	}
	Fail(c, http.StatusInternalServerError, model.CodeInternal, msg)
}

func Unavailable(c *gin.Context, msg string) {
	if msg == "" {
		msg = "依赖不可用"
	}
	Fail(c, http.StatusServiceUnavailable, model.CodeUnavailable, msg)
}

func mapStoreErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		NotFound(c, "资源不存在")
		return true
	}
	if errors.Is(err, store.ErrConflict) {
		Conflict(c, "资源冲突（规则名重复等）")
		return true
	}
	Internal(c, "内部错误")
	return true
}

func formatRule(r *model.Rule) gin.H {
	return gin.H{
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

func formatTask(t *model.Task) gin.H {
	return gin.H{
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

func formatResult(r *model.CrawlResult) gin.H {
	return gin.H{
		"id":         r.ID,
		"task_id":    r.TaskID,
		"url":        r.URL,
		"payload":    r.Payload,
		"created_at": timeutil.FormatNaive(r.CreatedAt),
	}
}

func formatNode(n model.WorkerNode) gin.H {
	return gin.H{
		"id":            n.ID,
		"role":          n.Role,
		"cpu":           n.CPU,
		"memory_mb":     n.MemoryMB,
		"pages_per_min": n.PagesPerMin,
		"fail_rate":     n.FailRate,
		"status":        n.Status,
		"last_seen":     timeutil.FormatNaive(n.LastSeen),
	}
}

func parseID(c *gin.Context, name string) (int64, bool) {
	var id int64
	if _, err := parseInt64(c.Param(name), &id); err != nil || id <= 0 {
		BadRequest(c, "无效的 id")
		return 0, false
	}
	return id, true
}

func parseInt64(s string, dest *int64) (int64, error) {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errBadID
		}
		n = n*10 + int64(ch-'0')
	}
	*dest = n
	return n, nil
}

var errBadID = errors.New("bad id")
