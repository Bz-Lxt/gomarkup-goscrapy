package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
	"goscrapy/internal/selector"
)

func (d *Deps) CreateSnapshot(c *gin.Context) {
	var req model.SnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		BadRequest(c, "url required")
		return
	}
	rec, err := d.Snap.Capture(c.Request.Context(), req.URL)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// Client disconnected or upstream cancelled: abort instead of writing
			// a 503 envelope, and do not perform further work.
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		Unavailable(c, "渲染器不可用")
		return
	}
	OK(c, model.SnapshotData{
		SnapshotID: rec.ID,
		Width:      rec.Width,
		Height:     rec.Height,
		ImageURL:   "/api/v1/snapshots/" + rec.ID + "/image",
		Nodes:      rec.Nodes,
	})
}

func (d *Deps) GetSnapshotImage(c *gin.Context) {
	id := c.Param("id")
	rec, ok := d.Snap.Get(id)
	if !ok {
		NotFound(c, "快照不存在")
		return
	}
	c.Data(http.StatusOK, "image/png", rec.PNG)
}

func (d *Deps) InferSelectors(c *gin.Context) {
	id := c.Param("id")
	rec, ok := d.Snap.Get(id)
	if !ok {
		NotFound(c, "快照不存在")
		return
	}
	var req model.SelectorRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.NodeID <= 0 {
		BadRequest(c, "node_id required")
		return
	}
	data, err := selector.Infer(rec.Tree, req.NodeID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, data)
}
