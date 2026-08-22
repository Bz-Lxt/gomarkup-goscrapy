package api

import (
	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
	"goscrapy/internal/store"
)

func (d *Deps) ListRules(c *gin.Context) {
	q := pageQuery(c)
	items, total, err := d.Repos.Rules.List(c.Request.Context(), q.Keyword, q.PageSize, q.Offset())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, formatRule(&items[i]))
	}
	OK(c, gin.H{"items": out, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func (d *Deps) CreateRule(c *gin.Context) {
	var in model.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		BadRequest(c, "参数校验失败")
		return
	}
	if err := in.ValidateCreate(); err != nil {
		BadRequest(c, err.Error())
		return
	}
	rule := &model.Rule{RespectRobots: true, QPS: 2, Version: 1}
	in.Apply(rule)
	if err := d.Repos.Rules.Create(c.Request.Context(), rule); err != nil {
		mapStoreErr(c, err)
		return
	}
	if !rule.RespectRobots {
		_ = d.Repos.Audit.RobotsDisabled(c.Request.Context(), rule.Name, rule.ID)
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionRuleCreated, rule.Name)
	Created(c, formatRule(rule))
}

func (d *Deps) GetRule(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	rule, err := d.Repos.Rules.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	OK(c, formatRule(rule))
}

func (d *Deps) PatchRule(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	rule, err := d.Repos.Rules.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	var in model.RuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		BadRequest(c, "参数校验失败")
		return
	}
	prevRobots := rule.RespectRobots
	in.Apply(rule)
	if err := d.Repos.Rules.Update(c.Request.Context(), rule); err != nil {
		mapStoreErr(c, err)
		return
	}
	if prevRobots && !rule.RespectRobots {
		_ = d.Repos.Audit.RobotsDisabled(c.Request.Context(), rule.Name, rule.ID)
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionRuleUpdated, rule.Name)
	OK(c, formatRule(rule))
}

func (d *Deps) DeleteRule(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	rule, err := d.Repos.Rules.Get(c.Request.Context(), id)
	if err != nil && err != store.ErrNotFound {
		// still try delete
	}
	if err := d.Repos.Rules.Delete(c.Request.Context(), id); err != nil {
		mapStoreErr(c, err)
		return
	}
	name := ""
	if rule != nil {
		name = rule.Name
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionRuleDeleted, name)
	OK(c, gin.H{"id": id})
}

func (d *Deps) PreviewRule(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	rule, err := d.Repos.Rules.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	var req model.PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数校验失败")
		return
	}
	if req.HTML == "" {
		BadRequest(c, "html required")
		return
	}
	out, err := d.Engine.Preview(rule, req.URL, req.HTML)
	if err != nil {
		Internal(c, err.Error())
		return
	}
	OK(c, gin.H{"items": out.Items, "links": out.Links})
}
