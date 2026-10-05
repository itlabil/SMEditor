package settings

import (
	"github.com/gin-gonic/gin"

	"smeditor/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/settings")
	g.GET("", h.get)
	g.PUT("", h.update)
	g.GET("/check", h.check)
}

func (h *Handler) get(c *gin.Context) {
	values, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, values)
}

func (h *Handler) update(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, httpx.ErrBadRequest("invalid_body", "Data yang dikirim tidak valid"))
		return
	}

	values, err := h.svc.Update(c.Request.Context(), body)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, values)
}

func (h *Handler) check(c *gin.Context) {
	results, err := h.svc.Check(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, results)
}
