package auth

import (
	"errors"
	"net/http"

	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(group *gin.RouterGroup, service *Service, authMiddleware gin.HandlerFunc) {
	h := &Handler{service: service}

	group.POST("/auth/register", h.Register)
	group.POST("/auth/login", h.Login)
	group.GET("/auth/me", authMiddleware, h.Me)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	result, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "register failed", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "register success", result)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	result, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "login failed", "invalid email or password")
			return
		}
		response.Error(c, http.StatusInternalServerError, "login failed", err.Error())
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
