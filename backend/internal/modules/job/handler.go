package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"smeditor/internal/httpx"
)

const ssePingInterval = 15 * time.Second

type Handler struct {
	svc *Service
	hub *Hub
}

func NewHandler(svc *Service, hub *Hub) *Handler {
	return &Handler{svc: svc, hub: hub}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	r.Group("/jobs").POST("/:id/cancel", h.cancel)
	r.Group("/projects").GET("/:id/events", h.events)
}

func (h *Handler) cancel(c *gin.Context) {
	if err := h.svc.Cancel(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) events(c *gin.Context) {
	projectID := c.Param("id")
	ch, unsubscribe := h.hub.Subscribe(projectID)
	defer unsubscribe()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(200)

	if last, err := h.svc.LatestEventForProject(c.Request.Context(), projectID); err == nil && last != nil {
		writeSSEEvent(c, *last)
	}
	c.Writer.Flush()

	ticker := time.NewTicker(ssePingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev := <-ch:
			writeSSEEvent(c, ev)
			c.Writer.Flush()
		case <-ticker.C:
			fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		}
	}
}

func writeSSEEvent(c *gin.Context, ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
}
