package auth

import (
	"errors"
	"net/http"
	"stylemind/internal/errs"
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
	response.Success(c, http.StatusCreated, "register success", result)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	result, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		response.Error(c, http.StatusInternalServerError, "login failed")
		return
	}
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
		response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	expiresAt, ok := expiresAtValue.(time.Time)
	if !ok {
		response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
		return
	}

	if err := h.service.Logout(c.Request.Context(), jti, expiresAt); err != nil {
		if errors.Is(err, errs.ErrUnauthorized) {
			response.Error(c, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
			return
		}
		response.Error(c, http.StatusServiceUnavailable, "logout failed")
		return
	}

	response.Success(c, http.StatusOK, "logout success", nil)
}
