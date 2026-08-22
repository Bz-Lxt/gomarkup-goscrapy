package api

import (
	"errors"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/auth"
	"goscrapy/internal/model"
)

func (d *Deps) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数校验失败")
		return
	}
	token, exp, err := d.Auth.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrBadCredential) {
			Fail(c, 401, model.CodeBadCredential, "用户名或密码错误")
			return
		}
		Internal(c, "内部错误")
		return
	}
	OK(c, model.LoginData{Token: token, ExpiresIn: exp, Username: auth.DefaultUser})
}

func (d *Deps) Health(c *gin.Context) {
	OK(c, model.HealthData{
		Status: "ok",
		Role:   d.Cfg.Role,
		Time:   model.NowString(),
	})
}
