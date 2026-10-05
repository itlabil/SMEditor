package prompt

import (
	"github.com/gin-gonic/gin"

	"smeditor/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	r.GET("/game-modes", h.listGameModes)

	g := r.Group("/prompt-blocks")
	g.GET("/:code", h.getBlock)
	g.PUT("/:code", h.updateBlock)
	g.POST("/:code/reset", h.resetBlock)
}

func (h *Handler) listGameModes(c *gin.Context) {
	list, err := h.svc.ListGameModes(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *Handler) getBlock(c *gin.Context) {
	b, err := h.svc.GetBlock(c.Request.Context(), c.Param("code"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, b)
}

func (h *Handler) updateBlock(c *gin.Context) {
	var req UpdateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrBadRequest("invalid_body", "Data yang dikirim tidak valid"))
		return
	}
	b, err := h.svc.UpdateBlock(c.Request.Context(), c.Param("code"), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, b)
}

func (h *Handler) resetBlock(c *gin.Context) {
	b, err := h.svc.ResetBlock(c.Request.Context(), c.Param("code"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, b)
}
