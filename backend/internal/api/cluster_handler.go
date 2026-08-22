package api

import (
	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

func (d *Deps) ClusterNodes(c *gin.Context) {
	nodes, err := d.Repos.Nodes.List(c.Request.Context())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	out := make([]gin.H, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, formatNode(n))
	}
	OK(c, gin.H{"items": out})
}

func (d *Deps) ClusterMetrics(c *gin.Context) {
	nodes, err := d.Repos.Nodes.List(c.Request.Context())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	st, err := d.Queue.Stats(c.Request.Context())
	if err != nil {
		Unavailable(c, "队列不可用")
		return
	}
	var ppm, fail float64
	if len(nodes) > 0 {
		var fsum float64
		for _, n := range nodes {
			ppm += n.PagesPerMin
			fsum += n.FailRate
		}
		fail = fsum / float64(len(nodes))
	}
	OK(c, model.ClusterMetrics{
		Time:        timeutil.Format(timeutil.Now()),
		Leader:      d.Elector != nil && d.Elector.IsLeader(),
		InstanceID:  d.Cfg.InstanceID,
		Nodes:       nodes,
		Queue:       st,
		PagesPerMin: ppm,
		FailRate:    fail,
	})
}

func (d *Deps) Proxies(c *gin.Context) {
	OK(c, gin.H{
		"mode":      d.Proxy.Mode(),
		"items":     d.Proxy.Snapshot(),
		"evictions": d.Proxy.Evictions(),
	})
}

func (d *Deps) QueueStats(c *gin.Context) {
	st, err := d.Queue.Stats(c.Request.Context())
	if err != nil {
		Unavailable(c, "队列不可用")
		return
	}
	OK(c, st)
}

func (d *Deps) WSMetrics(c *gin.Context) {
	d.Hub.ServeHTTP(c.Writer, c.Request)
}
