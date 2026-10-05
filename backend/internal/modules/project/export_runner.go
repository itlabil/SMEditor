package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"smeditor/internal/modules/job"
	"smeditor/internal/storage"
)

// exportTempSuffix marks a copy still in progress, so a canceled or
// failed export never leaves a truncated file under its final name, and
// a re-run knows to remove it first, per .agents/skills/sm-job-worker
// ("Runner harus aman diulang").
const exportTempSuffix = ".smeditor-tmp"

// exportCopyBufSize is the chunk size used while copying, small enough
// to check ctx and report progress often, large enough not to make
// copying a multi-GB video slow.
const exportCopyBufSize = 1 << 20 // 1 MiB

// exportFile is one file "Salin ke folder" copies: src is its fixed
// name inside data/projects/<id>/, dst its project-named name in the
// target folder (SM-17).
type exportFile struct {
	src, dst string
	// rewriteVideo marks highlight.json, whose "video" field must name
	// the renamed video instead of source.mp4.
	rewriteVideo bool
}

func exportFilesFor(p *Project) []exportFile {
	videoFile := p.VideoFile
	if videoFile == "" {
		videoFile = storage.SourceVideoFile
	}
	names := ExportFileNamesFor(p.Name, videoFile)
	return []exportFile{
		{src: videoFile, dst: names.Video},
		{src: storage.HighlightFile, dst: names.Highlight, rewriteVideo: true},
		{src: storage.NarrationFile, dst: names.Narasi},
	}
}

// ExportRunner implements job.Runner for job.TypeExport: it copies a
// project's finished output into j.Payload, an absolute folder computed
// by Service.ExportToFolder (the user's chosen destination plus the
// project's sanitized name). It never touches the project's status (see
// project.JobSync, whose switches simply don't list TypeExport).
type ExportRunner struct {
	repo    *Repository
	storage *storage.Storage
}

func NewExportRunner(repo *Repository, st *storage.Storage) *ExportRunner {
	return &ExportRunner{repo: repo, storage: st}
}

func (r *ExportRunner) Type() string { return job.TypeExport }

func (r *ExportRunner) Run(ctx context.Context, j job.Job, report job.ProgressFunc) error {
	p, err := r.repo.FindByID(ctx, j.ProjectID)
	if err != nil {
		return fmt.Errorf("baca project: %w", err)
	}
	files := exportFilesFor(p)

	targetDir := j.Payload
	if targetDir == "" {
		return fmt.Errorf("folder tujuan export kosong")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("buat folder tujuan: %w", err)
	}

	var totalSize int64
	sizes := make(map[string]int64, len(files))
	for _, f := range files {
		srcPath, err := r.storage.FilePath(j.ProjectID, f.src)
		if err != nil {
			return err
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("baca %s: %w", f.src, err)
		}
		sizes[f.src] = info.Size()
		totalSize += info.Size()
	}

	// A leftover temp file from an interrupted previous attempt must be
	// removed before copying starts again.
	for _, f := range files {
		_ = os.Remove(filepath.Join(targetDir, f.dst+exportTempSuffix))
	}

	var copiedBefore int64
	for _, f := range files {
		srcPath, err := r.storage.FilePath(j.ProjectID, f.src)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(targetDir, f.dst)
		tmpPath := dstPath + exportTempSuffix

		if f.rewriteVideo {
			err = writeHighlightCopy(srcPath, tmpPath, files[0].dst)
		} else {
			err = copyFileWithProgress(ctx, srcPath, tmpPath, func(copiedThisFile int64) {
				percent := 100.0
				if totalSize > 0 {
					percent = float64(copiedBefore+copiedThisFile) / float64(totalSize) * 100
				}
				report(percent, "Menyalin "+f.dst)
			})
		}
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("salin %s: %w", f.dst, err)
		}
		if err := os.Rename(tmpPath, dstPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("timpa %s: %w", f.dst, err)
		}
		copiedBefore += sizes[f.src]
	}

	report(100, "Selesai")
	return nil
}

// writeHighlightCopy writes highlight.json to dstPath with its "video"
// field set to video, the renamed video file next to it.
func writeHighlightCopy(srcPath, dstPath, video string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	rewritten, err := RewriteHighlightVideo(data, video)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, rewritten, 0o644)
}

// copyFileWithProgress copies srcPath to dstPath (overwriting dstPath if
// it already exists), calling onBytes after every chunk with the total
// bytes copied so far for this file, and stopping as soon as ctx is
// canceled.
func copyFileWithProgress(ctx context.Context, srcPath, dstPath string, onBytes func(copiedSoFar int64)) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, exportCopyBufSize)
	var copied int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			copied += int64(n)
			onBytes(copied)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return dst.Sync()
}
