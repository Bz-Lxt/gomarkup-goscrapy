package api

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"goscrapy/internal/auth"
	"goscrapy/internal/bloom"
	"goscrapy/internal/config"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/parser"
	"goscrapy/internal/proxy"
	"goscrapy/internal/queue"
	"goscrapy/internal/renderer"
	"goscrapy/internal/scheduler"
	"goscrapy/internal/store"
	"goscrapy/internal/ws"
)

type Deps struct {
	Cfg     *config.Config
	Auth    *auth.Service
	Repos   *store.Repos
	Redis   *redis.Client
	Queue   *queue.Queue
	Bloom   *bloom.Filter
	Proxy   *proxy.Pool
	Fetch   *fetcher.Client
	Engine  *parser.Engine
	Snap    *renderer.Service
	Hub     *ws.Hub
	Elector *scheduler.Elector
}

func pageQuery(c *gin.Context) modelPage {
	var q struct {
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
		Keyword  string `form:"keyword"`
		Status   string `form:"status"`
		TaskID   int64  `form:"task_id"`
	}
	_ = c.ShouldBindQuery(&q)
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	return modelPage{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword, Status: q.Status, TaskID: q.TaskID}
}

type modelPage struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
	TaskID   int64
}

func (q modelPage) Offset() int { return (q.Page - 1) * q.PageSize }
