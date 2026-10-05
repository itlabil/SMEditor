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

// exportFiles are the files "Salin ke folder" copies, per docs/prd.md.
// highlight.json and narasi.txt are copied byte-for-byte, so
// highlight.json's "video" field (already storage.SourceVideoFile)
// keeps matching the copied video's file name without any rewriting.
var exportFiles = []string{storage.SourceVideoFile, storage.HighlightFile, storage.NarrationFile}

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
	if _, err := r.repo.FindByID(ctx, j.ProjectID); err != nil {
		return fmt.Errorf("baca project: %w", err)
	}

	targetDir := j.Payload
	if targetDir == "" {
		return fmt.Errorf("folder tujuan export kosong")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("buat folder tujuan: %w", err)
	}

	var totalSize int64
	sizes := make(map[string]int64, len(exportFiles))
	for _, name := range exportFiles {
		srcPath, err := r.storage.FilePath(j.ProjectID, name)
		if err != nil {
			return err
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("baca %s: %w", name, err)
		}
		sizes[name] = info.Size()
		totalSize += info.Size()
	}

	// A leftover temp file from an interrupted previous attempt must be
	// removed before copying starts again.
	for _, name := range exportFiles {
		_ = os.Remove(filepath.Join(targetDir, name+exportTempSuffix))
	}

	var copiedBefore int64
	for _, name := range exportFiles {
		srcPath, err := r.storage.FilePath(j.ProjectID, name)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(targetDir, name)
		tmpPath := dstPath + exportTempSuffix

		err = copyFileWithProgress(ctx, srcPath, tmpPath, func(copiedThisFile int64) {
			percent := 100.0
			if totalSize > 0 {
				percent = float64(copiedBefore+copiedThisFile) / float64(totalSize) * 100
			}
			report(percent, "Menyalin "+name)
		})
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("salin %s: %w", name, err)
		}
		if err := os.Rename(tmpPath, dstPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("timpa %s: %w", name, err)
		}
		copiedBefore += sizes[name]
	}

	report(100, "Selesai")
	return nil
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
