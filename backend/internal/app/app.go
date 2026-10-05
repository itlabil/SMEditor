package app

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"smeditor/internal/db"
	"smeditor/internal/httpx"
	"smeditor/internal/webdist"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	registerFrontend(r)

	return r
}

// registerFrontend serves the built Vue app in production. In dev, the
// frontend is served by the Vite dev server, which proxies /api here.
func registerFrontend(r *gin.Engine) {
	if !webdist.Embedded {
		r.NoRoute(func(c *gin.Context) {
			c.String(http.StatusNotFound, "Frontend belum di-build. Jalankan `make dev` lalu akses lewat Vite dev server.")
		})
		return
	}

	fileServer := http.FileServer(http.FS(webdist.Dist))
	r.NoRoute(func(c *gin.Context) {
		if _, err := fs.Stat(webdist.Dist, c.Request.URL.Path[1:]); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// Run starts the HTTP server, bound to 127.0.0.1 only.
func Run() error {
	cfg := LoadConfig()

	conn, err := db.Open(filepath.Join(cfg.DataDir, "app.db"))
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := db.Migrate(context.Background(), conn); err != nil {
		return err
	}

	r := NewRouter()

	srv := &http.Server{
		Addr:    "127.0.0.1:" + cfg.Port,
		Handler: r,
	}

	log.Printf("smeditor listening on %s", srv.Addr)
	return srv.ListenAndServe()
}
