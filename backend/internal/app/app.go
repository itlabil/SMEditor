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
	"smeditor/internal/modules/job"
	"smeditor/internal/modules/project"
	"smeditor/internal/modules/settings"
	"smeditor/internal/storage"
	"smeditor/internal/tools"
	"smeditor/internal/tools/ffmpeg"
	"smeditor/internal/tools/whisper"
	"smeditor/internal/tools/ytdlp"
	"smeditor/internal/webdist"
)

// NewRouter wires every module and starts the job worker. ctx governs the
// worker's background goroutine, per .agents/skills/sm-job-worker; it
// should live as long as the server does.
func NewRouter(ctx context.Context, conn *sql.DB, st *storage.Storage) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})

	settingsRepo := settings.NewRepository(conn)
	settingsSvc := settings.NewService(settingsRepo, ytdlp.Client{}, ffmpeg.Ffmpeg{}, ffmpeg.Ffprobe{}, whisper.Client{})
	settings.NewHandler(settingsSvc).Register(api)

	jobRepo := job.NewRepository(conn)
	jobHub := job.NewHub()
	jobWorker := job.NewWorker(jobRepo, jobHub, nil)
	jobSvc := job.NewService(jobRepo, jobWorker)
	job.NewHandler(jobSvc, jobHub).Register(api)

	if err := jobWorker.RecoverStaleRunning(ctx); err != nil {
		return nil, err
	}
	jobWorker.Start(ctx)

	projectRepo := project.NewRepository(conn)
	projectSvc := project.NewService(projectRepo, st, jobSvc)
	project.NewHandler(projectSvc).Register(api)

	registerFrontend(r)

	return r, nil
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

	ctx := context.Background()
	if err := db.Migrate(ctx, conn); err != nil {
		return err
	}

	st := storage.New(cfg.DataPath())
	r, err := NewRouter(ctx, conn, st)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    "127.0.0.1:" + cfg.Port,
		Handler: r,
	}

	log.Printf("smeditor listening on %s (base dir: %s)", srv.Addr, cfg.BaseDir)
	return srv.ListenAndServe()
}
