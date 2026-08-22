package api

import "github.com/gin-gonic/gin"

func (d *Deps) ListTaskResults(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := d.Repos.Tasks.Get(c.Request.Context(), id); err != nil {
		mapStoreErr(c, err)
		return
	}
	q := pageQuery(c)
	items, total, err := d.Repos.Results.ListByTask(c.Request.Context(), id, q.PageSize, q.Offset())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, formatResult(&items[i]))
	}
	OK(c, gin.H{"items": out, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func (d *Deps) ListResults(c *gin.Context) {
	q := pageQuery(c)
	items, total, err := d.Repos.Results.List(c.Request.Context(), q.TaskID, q.PageSize, q.Offset())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, formatResult(&items[i]))
	}
	OK(c, gin.H{"items": out, "total": total, "page": q.Page, "page_size": q.PageSize})
}
