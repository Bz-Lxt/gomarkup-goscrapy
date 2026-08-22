package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/model"
)

const ContextUser = "auth.username"

func Middleware(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		claims, err := svc.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Envelope{
				Code:    model.CodeUnauthorized,
				Message: "未登录或令牌失效",
				Data:    map[string]any{},
			})
			return
		}
		c.Set(ContextUser, claims.Username)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); h != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(h, prefix) {
			return strings.TrimSpace(h[len(prefix):])
		}
		return strings.TrimSpace(h)
	}
	if q := strings.TrimSpace(c.Query("token")); q != "" {
		return q
	}
	return ""
}

func Username(c *gin.Context) string {
	v, _ := c.Get(ContextUser)
	s, _ := v.(string)
	return s
}
