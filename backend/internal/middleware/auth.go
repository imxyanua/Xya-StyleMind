package middleware

import (
	"context"
	"net/http"
	"strings"

	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type JWTAuthOptions struct {
	revocationStore auth.TokenRevocationStore
	statusChecker   UserStatusChecker
}

type JWTAuthOption func(*JWTAuthOptions)

type UserStatusChecker interface {
	GetUserByID(ctx context.Context, id string) (*auth.User, error)
}

func WithTokenRevocationStore(store auth.TokenRevocationStore) JWTAuthOption {
	return func(o *JWTAuthOptions) {
		o.revocationStore = store
	}
}

func WithUserStatusChecker(checker UserStatusChecker) JWTAuthOption {
	return func(o *JWTAuthOptions) {
		o.statusChecker = checker
	}
}

func JWTAuth(tokenConfig auth.TokenConfig, opts ...JWTAuthOption) gin.HandlerFunc {
	options := JWTAuthOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(tokenConfig, parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
			c.Abort()
			return
		}
		if options.revocationStore != nil {
			revoked, err := options.revocationStore.IsTokenRevoked(c.Request.Context(), claims.ID)
			if err != nil || revoked {
				response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				c.Abort()
				return
			}
		}
		if options.statusChecker != nil {
			user, err := options.statusChecker.GetUserByID(c.Request.Context(), claims.UserID)
			if err != nil || user == nil || user.Status == "disabled" {
				response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("token_jti", claims.ID)
		if claims.ExpiresAt != nil {
			c.Set("token_expires_at", claims.ExpiresAt.Time)
		}
		c.Next()
	}
}
