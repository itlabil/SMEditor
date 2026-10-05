package highlight

import (
	"github.com/gin-gonic/gin"

	"smeditor/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/projects")
	g.POST("/:id/highlight", h.save)
	g.GET("/:id/highlight", h.get)
	g.DELETE("/:id/highlight", h.delete)
	g.GET("/:id/narasi", h.narasi)
}

type saveRequest struct {
	Body string `json:"body"`
}

// response adds display-only computed fields on top of SavedHighlight,
// without polluting the on-disk highlight.json format. Peringatan flags
// segments with an unusually short/long duration; unlike validation
// errors, these never block saving (see Warnings).
type response struct {
	SavedHighlight
	TotalDurasiSec float64           `json:"total_durasi_sec"`
	Peringatan     []ValidationError `json:"peringatan"`
}

func toResponse(saved *SavedHighlight) response {
	return response{
		SavedHighlight: *saved,
		TotalDurasiSec: TotalDuration(saved.Segmen),
		Peringatan:     Warnings(&Highlight{Segmen: saved.Segmen}),
	}
}

func (h *Handler) save(c *gin.Context) {
	var req saveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrBadRequest("invalid_body", "Data yang dikirim tidak valid"))
		return
	}
	saved, err := h.svc.Save(c.Request.Context(), c.Param("id"), req.Body)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(saved))
}

func (h *Handler) get(c *gin.Context) {
	saved, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(saved))
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) narasi(c *gin.Context) {
	path, err := h.svc.NarasiPath(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.FileAttachment(path, "narasi.txt")
}
