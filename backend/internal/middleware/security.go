package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	headerContentTypeOptions = "X-Content-Type-Options"
	headerFrameOptions       = "X-Frame-Options"
	headerReferrerPolicy     = "Referrer-Policy"
	headerCSP                = "Content-Security-Policy"
	headerPermissionsPolicy  = "Permissions-Policy"
	headerCacheControl       = "Cache-Control"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(headerContentTypeOptions, "nosniff")
		c.Header(headerFrameOptions, "DENY")
		c.Header(headerReferrerPolicy, "no-referrer")
		c.Header(headerCSP, "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		c.Header(headerPermissionsPolicy, "camera=(), microphone=(), geolocation=(), payment=()")

		if shouldDisableCache(c.Request.URL.Path) {
			c.Header(headerCacheControl, "no-store")
		}

		c.Next()
	}
}

func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 || c.Request.Body == nil || c.Request.Body == http.NoBody {
			c.Next()
			return
		}

		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBytes+1))
		_ = c.Request.Body.Close()
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid request body")
			c.Abort()
			return
		}
		if int64(len(body)) > maxBytes {
			response.Error(c, http.StatusRequestEntityTooLarge, "request body too large")
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Next()
	}
}

func shouldDisableCache(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth") || strings.HasPrefix(path, "/api/v1/admin")
}
