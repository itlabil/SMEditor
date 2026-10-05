package project

import (
	"github.com/gin-gonic/gin"

	"smeditor/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/projects")
	g.GET("", h.list)
	g.POST("", h.create)
	g.GET("/:id", h.get)
	g.DELETE("/:id", h.delete)
	g.GET("/:id/thumbnail", h.thumbnail)
	g.POST("/:id/download", h.retryDownload)
	g.POST("/:id/transcribe", h.retryTranscribe)
	g.GET("/:id/transcript", h.transcript)
	g.GET("/:id/prompt", h.prompt)
	g.POST("/:id/open-folder", h.openFolder)
}

func (h *Handler) list(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *Handler) create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrBadRequest("invalid_body", "Data yang dikirim tidak valid"))
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, p)
}

func (h *Handler) get(c *gin.Context) {
	p, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, p)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) thumbnail(c *gin.Context) {
	path, err := h.svc.ThumbnailPath(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.File(path)
}

func (h *Handler) retryDownload(c *gin.Context) {
	p, err := h.svc.RetryDownload(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, p)
}

type retryTranscribeRequest struct {
	TranscriptLang string `json:"transcript_lang"`
}

func (h *Handler) retryTranscribe(c *gin.Context) {
	var req retryTranscribeRequest
	// Body is optional: omitting it just re-runs with the current language.
	_ = c.ShouldBindJSON(&req)

	p, err := h.svc.RetryTranscribe(c.Request.Context(), c.Param("id"), req.TranscriptLang)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, p)
}

func (h *Handler) transcript(c *gin.Context) {
	path, filename, err := h.svc.TranscriptPath(c.Request.Context(), c.Param("id"), c.Query("format"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.FileAttachment(path, filename)
}

func (h *Handler) prompt(c *gin.Context) {
	text, err := h.svc.Prompt(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"prompt": text})
}

func (h *Handler) openFolder(c *gin.Context) {
	if err := h.svc.OpenFolder(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
