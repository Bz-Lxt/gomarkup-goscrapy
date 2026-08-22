package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"goscrapy/internal/auth"
	"goscrapy/internal/logger"
)

func NewRouter(d *Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLog())
	r.Use(cors())

	v1 := r.Group("/api/v1")
	v1.GET("/health", d.Health)
	v1.POST("/auth/login", d.Login)

	authed := v1.Group("")
	authed.Use(auth.Middleware(d.Auth))
	authed.GET("/rules", d.ListRules)
	authed.POST("/rules", d.CreateRule)
	authed.GET("/rules/:id", d.GetRule)
	authed.PATCH("/rules/:id", d.PatchRule)
	authed.DELETE("/rules/:id", d.DeleteRule)
	authed.POST("/rules/:id/preview", d.PreviewRule)

	authed.POST("/tasks", d.CreateTask)
	authed.GET("/tasks", d.ListTasks)
	authed.GET("/tasks/:id", d.GetTask)
	authed.POST("/tasks/:id/start", d.StartTask)
	authed.POST("/tasks/:id/pause", d.PauseTask)
	authed.POST("/tasks/:id/cancel", d.CancelTask)
	authed.GET("/tasks/:id/results", d.ListTaskResults)
	authed.GET("/results", d.ListResults)

	authed.POST("/snapshots", d.CreateSnapshot)
	authed.GET("/snapshots/:id/image", d.GetSnapshotImage)
	authed.POST("/snapshots/:id/selectors", d.InferSelectors)

	authed.GET("/cluster/nodes", d.ClusterNodes)
	authed.GET("/cluster/metrics", d.ClusterMetrics)
	authed.GET("/proxies", d.Proxies)
	authed.GET("/queue/stats", d.QueueStats)
	authed.GET("/ws/metrics", d.WSMetrics)

	r.NoRoute(func(c *gin.Context) {
		NotFound(c, "资源不存在")
	})
	return r
}

func requestLog() gin.HandlerFunc {
	lg := logger.Named("http")
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		lg.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("dur", time.Since(start)),
		)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
