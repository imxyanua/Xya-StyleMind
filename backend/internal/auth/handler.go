package auth

import (
	"errors"
	"net/http"
	"strings"
	"stylemind/internal/errs"
	"stylemind/pkg/logger"
	"time"

	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(group *gin.RouterGroup, service *Service, authMiddleware gin.HandlerFunc, loginRateLimit gin.HandlerFunc, registerRateLimit gin.HandlerFunc) {
	h := &Handler{service: service}

	group.POST("/auth/register", registerRateLimit, h.Register)
	group.POST("/auth/login", loginRateLimit, h.Login)
	group.GET("/auth/me", authMiddleware, h.Me)
	group.POST("/auth/logout", authMiddleware, h.Logout)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	result, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, errs.ErrEmailAlreadyExists) {
			response.Error(c, http.StatusBadRequest, "email already exists")
			return
		}
		response.Error(c, http.StatusBadRequest, "register failed")
		return
	}
	logger.Audit(c, "auth.register", logger.AuditResultSuccess, map[string]any{
		"email":   normalizedEmail(req.Email),
		"user_id": authResultUserID(result),
		"role":    authResultRole(result),
	})
	response.Success(c, http.StatusCreated, "register success", result)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Audit(c, "auth.login", logger.AuditResultFailed, map[string]any{
			"reason": "invalid_payload",
		})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		logger.Audit(c, "auth.login", logger.AuditResultFailed, map[string]any{
			"email":  normalizedEmail(req.Email),
			"reason": "validation_error",
		})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	result, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			logger.Audit(c, "auth.login", logger.AuditResultFailed, map[string]any{
				"email":  normalizedEmail(req.Email),
				"reason": "invalid_credentials",
			})
			response.Error(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		if errors.Is(err, errs.ErrUserDisabled) {
			logger.Audit(c, "auth.login", logger.AuditResultFailed, map[string]any{
				"email":  normalizedEmail(req.Email),
				"reason": "account_disabled",
			})
			response.Error(c, http.StatusForbidden, "account disabled")
			return
		}
		logger.Audit(c, "auth.login", logger.AuditResultFailed, map[string]any{
			"email":  normalizedEmail(req.Email),
			"reason": "internal_error",
		})
		response.Error(c, http.StatusInternalServerError, "login failed")
		return
	}
	logger.Audit(c, "auth.login", logger.AuditResultSuccess, map[string]any{
		"email":   normalizedEmail(req.Email),
		"user_id": authResultUserID(result),
		"role":    authResultRole(result),
	})
	response.Success(c, http.StatusOK, "login success", result)
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")
	response.Success(c, http.StatusOK, "authorized", gin.H{
		"user_id": userID,
		"role":    role,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	jti := c.GetString("token_jti")
	expiresAtValue, ok := c.Get("token_expires_at")
	if !ok {
		logger.Audit(c, "auth.logout", logger.AuditResultFailed, map[string]any{
			"reason": "unauthorized",
		})
		response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	expiresAt, ok := expiresAtValue.(time.Time)
	if !ok {
		logger.Audit(c, "auth.logout", logger.AuditResultFailed, map[string]any{
			"reason": "unauthorized",
		})
		response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	if err := h.service.Logout(c.Request.Context(), jti, expiresAt); err != nil {
		if errors.Is(err, errs.ErrUnauthorized) {
			logger.Audit(c, "auth.logout", logger.AuditResultFailed, map[string]any{
				"reason": "unauthorized",
			})
			response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
			return
		}
		logger.Audit(c, "auth.logout", logger.AuditResultFailed, map[string]any{
			"reason": "revocation_store_error",
		})
		response.Error(c, http.StatusServiceUnavailable, "logout failed")
		return
	}

	logger.Audit(c, "auth.logout", logger.AuditResultSuccess, nil)
	response.Success(c, http.StatusOK, "logout success", nil)
}

func normalizedEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func authResultUserID(result map[string]interface{}) string {
	user, ok := result["user"].(map[string]interface{})
	if !ok {
		return ""
	}
	id, _ := user["id"].(string)
	return id
}

func authResultRole(result map[string]interface{}) string {
	user, ok := result["user"].(map[string]interface{})
	if !ok {
		return ""
	}
	role, _ := user["role"].(string)
	return role
}
