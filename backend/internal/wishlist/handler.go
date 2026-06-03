package wishlist

import (
	"errors"
	"net/http"
	"stylemind/internal/errs"

	"stylemind/pkg/pagination"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}

	wishlist := api.Group("/wishlist")
	wishlist.Use(authMiddleware)
	wishlist.GET("", h.List)
	wishlist.POST("/products/:product_id", h.AddProduct)
	wishlist.DELETE("/products/:product_id", h.RemoveProduct)
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	page := pagination.Parse(c)
	items, total, err := h.service.List(c.Request.Context(), userID, page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch wishlist")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) AddProduct(c *gin.Context) {
	userID := c.GetString("user_id")
	productID := c.Param("product_id")

	if err := h.service.AddProduct(c.Request.Context(), userID, productID); err != nil {
		h.writeProductMutationError(c, err, "failed to add wishlist product")
		return
	}

	response.Success(c, http.StatusOK, "wishlist product added", gin.H{"product_id": productID})
}

func (h *Handler) RemoveProduct(c *gin.Context) {
	userID := c.GetString("user_id")
	productID := c.Param("product_id")

	if err := h.service.RemoveProduct(c.Request.Context(), userID, productID); err != nil {
		h.writeProductMutationError(c, err, "failed to remove wishlist product")
		return
	}

	response.Success(c, http.StatusOK, "wishlist product removed", gin.H{"product_id": productID})
}

func (h *Handler) writeProductMutationError(c *gin.Context, err error, fallback string) {
	if errors.Is(err, errs.ErrInvalidID) {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}
	if errors.Is(err, errs.ErrProductNotFound) {
		response.Error(c, http.StatusNotFound, "product not found")
		return
	}
	response.Error(c, http.StatusInternalServerError, fallback)
}
