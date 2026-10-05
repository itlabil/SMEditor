package app

import (
	"context"
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"smeditor/internal/db"
	"smeditor/internal/httpx"
	"smeditor/internal/modules/settings"
	"smeditor/internal/tools"
	"smeditor/internal/tools/ffmpeg"
	"smeditor/internal/tools/whisper"
	"smeditor/internal/tools/ytdlp"
	"smeditor/internal/webdist"
)

func NewRouter(conn *sql.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	settingsRepo := settings.NewRepository(conn)
	settingsSvc := settings.NewService(settingsRepo, ytdlp.Client{}, ffmpeg.Ffmpeg{}, ffmpeg.Ffprobe{}, whisper.Client{})
	settings.NewHandler(settingsSvc).Register(api)

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
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	tools.BaseDir = cfg.BaseDir

	conn, err := db.Open(filepath.Join(cfg.DataPath(), "app.db"))
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := db.Migrate(context.Background(), conn); err != nil {
		return err
	}

	r := NewRouter(conn)

	srv := &http.Server{
		Addr:    "127.0.0.1:" + cfg.Port,
		Handler: r,
	}

	log.Printf("smeditor listening on %s (base dir: %s)", srv.Addr, cfg.BaseDir)
	return srv.ListenAndServe()
}
