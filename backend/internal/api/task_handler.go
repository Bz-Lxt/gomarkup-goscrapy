package api

import (
	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
	"goscrapy/internal/scheduler"
)

func (d *Deps) CreateTask(c *gin.Context) {
	var in model.TaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		BadRequest(c, "参数校验失败")
		return
	}
	in = in.Normalize()
	if err := in.Validate(); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if _, err := d.Repos.Rules.Get(c.Request.Context(), in.RuleID); err != nil {
		mapStoreErr(c, err)
		return
	}
	task := &model.Task{
		Name:        in.Name,
		RuleID:      in.RuleID,
		SeedURLs:    in.SeedURLs,
		MaxDepth:    in.MaxDepth,
		Concurrency: in.Concurrency,
		Status:      model.TaskCreated,
		Stats:       model.JSONMap{},
	}
	if err := d.Repos.Tasks.Create(c.Request.Context(), task); err != nil {
		Internal(c, "内部错误")
		return
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionTaskCreated, task.Name)
	Accepted(c, formatTask(task))
}

func (d *Deps) ListTasks(c *gin.Context) {
	q := pageQuery(c)
	items, total, err := d.Repos.Tasks.List(c.Request.Context(), q.Status, q.PageSize, q.Offset())
	if err != nil {
		Internal(c, "内部错误")
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, formatTask(&items[i]))
	}
	OK(c, gin.H{"items": out, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func (d *Deps) GetTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	task, err := d.Repos.Tasks.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	OK(c, formatTask(task))
}

func (d *Deps) StartTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	task, err := d.Repos.Tasks.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	if !task.CanStart() {
		BadRequest(c, "当前状态不可启动: "+task.Status)
		return
	}
	rule, err := d.Repos.Rules.Get(c.Request.Context(), task.RuleID)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	wasCreated := task.Status == model.TaskCreated
	updated, err := d.Repos.Tasks.UpdateStatus(c.Request.Context(), id, []string{model.TaskCreated, model.TaskPaused}, model.TaskRunning)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	if wasCreated {
		if _, err := scheduler.EnqueueSeeds(c.Request.Context(), d.Queue, d.Bloom, d.Repos.Tasks, updated, rule); err != nil {
			_, _ = d.Repos.Tasks.UpdateStatus(c.Request.Context(), id, []string{model.TaskRunning}, model.TaskCreated)
			Unavailable(c, "队列不可用")
			return
		}
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionTaskStarted, updated.Name)
	OK(c, formatTask(updated))
}

func (d *Deps) PauseTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	task, err := d.Repos.Tasks.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	if !task.CanPause() {
		BadRequest(c, "当前状态不可暂停: "+task.Status)
		return
	}
	updated, err := d.Repos.Tasks.UpdateStatus(c.Request.Context(), id, []string{model.TaskRunning}, model.TaskPaused)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionTaskPaused, updated.Name)
	OK(c, formatTask(updated))
}

func (d *Deps) CancelTask(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	task, err := d.Repos.Tasks.Get(c.Request.Context(), id)
	if err != nil {
		mapStoreErr(c, err)
		return
	}
	if !task.CanCancel() {
		BadRequest(c, "当前状态不可取消: "+task.Status)
		return
	}
	updated, err := d.Repos.Tasks.UpdateStatus(c.Request.Context(), id,
		[]string{model.TaskCreated, model.TaskRunning, model.TaskPaused}, model.TaskCancelled)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	if _, err := d.Queue.DropTask(c.Request.Context(), id); err != nil {
		Unavailable(c, "队列不可用")
		return
	}
	_ = d.Bloom.Drop(c.Request.Context(), id)
	_ = d.Repos.Audit.Append(c.Request.Context(), model.ActionTaskCancelled, updated.Name)
	OK(c, formatTask(updated))
}
