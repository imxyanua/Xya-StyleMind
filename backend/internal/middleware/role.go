package middleware

import (
	"net/http"
	"stylemind/internal/errs"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("user_role")
		if userRole != role {
			response.Error(c, http.StatusForbidden, errs.ErrForbidden.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}
