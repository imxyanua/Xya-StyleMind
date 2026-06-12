package address

import (
	"errors"
	"net/http"
	"stylemind/internal/errs"
	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}
	addresses := api.Group("/me/addresses")
	addresses.Use(authMiddleware)
	addresses.GET("", h.List)
	addresses.POST("", h.Create)
	addresses.PATCH("/:id", h.Update)
	addresses.DELETE("/:id", h.Delete)
	addresses.PATCH("/:id/default", h.SetDefault)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch addresses")
		return
	}
	response.Success(c, http.StatusOK, "ok", items)
}

func (h *Handler) Create(c *gin.Context) {
	var req AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	item, err := h.service.Create(c.Request.Context(), c.GetString("user_id"), req)
	if err != nil {
		h.writeMutationError(c, err, "failed to create address")
		return
	}
	response.Success(c, http.StatusCreated, "address created", item)
}

func (h *Handler) Update(c *gin.Context) {
	var req AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req)
	if err != nil {
		h.writeMutationError(c, err, "failed to update address")
		return
	}
	response.Success(c, http.StatusOK, "address updated", item)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.GetString("user_id"), c.Param("id")); err != nil {
		h.writeMutationError(c, err, "failed to delete address")
		return
	}
	response.Success(c, http.StatusOK, "address deleted", gin.H{"id": c.Param("id")})
}

func (h *Handler) SetDefault(c *gin.Context) {
	item, err := h.service.SetDefault(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		h.writeMutationError(c, err, "failed to set default address")
		return
	}
	response.Success(c, http.StatusOK, "default address updated", item)
}

func (h *Handler) writeMutationError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid address id")
	case errors.Is(err, errs.ErrValidationFailed):
		response.Error(c, http.StatusBadRequest, "validation failed")
	case errors.Is(err, errs.ErrAddressNotFound):
		response.Error(c, http.StatusNotFound, "address not found")
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}
