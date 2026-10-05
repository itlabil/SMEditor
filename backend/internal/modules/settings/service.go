package settings

import (
	"context"
	"fmt"

	"smeditor/internal/httpx"
)

// ToolClient checks whether a configured tool path resolves to a real,
// runnable binary. Implementations live under internal/tools/*; each one
// wraps a single exec.Command call site.
type ToolClient interface {
	Check(ctx context.Context, path string) (resolvedPath string, found bool, version string)
}

// WhisperClient is a ToolClient that also reports, from the same
// --version run, which compute backend whisper-cli loaded ("gpu", "cpu",
// or "" if unknown) and the GPU name. Implemented by whisper.Client.
type WhisperClient interface {
	CheckWithBackend(ctx context.Context, path string) (resolvedPath string, found bool, version, backend, gpu string)
}

// ModelClient checks whether a configured model file exists. Unlike
// ToolClient, it never runs anything: a model file has no --version to
// call, so there is no "version" to report.
type ModelClient interface {
	CheckModel(ctx context.Context, path string) (resolvedPath string, found bool, version string)
}

type Service struct {
	repo         *Repository
	ytdlp        ToolClient
	ffmpeg       ToolClient
	ffprobe      ToolClient
	whisper      WhisperClient
	whisperModel ModelClient
}

func NewService(repo *Repository, ytdlp, ffmpeg, ffprobe ToolClient, whisper WhisperClient, whisperModel ModelClient) *Service {
	return &Service{repo: repo, ytdlp: ytdlp, ffmpeg: ffmpeg, ffprobe: ffprobe, whisper: whisper, whisperModel: whisperModel}
}

// Get returns every known setting key, filling in the default value for
// anything not yet saved.
func (s *Service) Get(ctx context.Context) (map[string]string, error) {
	stored, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(defaults))
	for key, value := range defaults {
		out[key] = value
	}
	for key, value := range stored {
		if _, known := defaults[key]; known {
			out[key] = value
		}
	}
	return out, nil
}

// Update saves only known setting keys; any other key is rejected so a typo
// in the request body never silently does nothing.
func (s *Service) Update(ctx context.Context, values map[string]string) (map[string]string, error) {
	clean := make(map[string]string, len(values))
	for key, value := range values {
		if _, known := defaults[key]; !known {
			return nil, httpx.ErrBadRequest("unknown_setting_key", fmt.Sprintf("Key pengaturan %q tidak dikenal", key))
		}
		if key == KeyWhisperDevice && !validWhisperDevices[value] {
			return nil, httpx.ErrBadRequest("invalid_whisper_device", "whisper_device harus auto, cpu, atau gpu")
		}
		clean[key] = value
	}

	if err := s.repo.Upsert(ctx, clean); err != nil {
		return nil, err
	}
	return s.Get(ctx)
}

// Check probes yt-dlp, ffmpeg, ffprobe, whisper, and the whisper model
// file using the currently saved (or default) paths.
func (s *Service) Check(ctx context.Context) ([]CheckResult, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	tools := []struct {
		name   string
		key    string
		client ToolClient
	}{
		{"yt-dlp", KeyYtdlpPath, s.ytdlp},
		{"ffmpeg", KeyFfmpegPath, s.ffmpeg},
		{"ffprobe", KeyFfprobePath, s.ffprobe},
	}

	results := make([]CheckResult, 0, len(tools)+2)
	for _, t := range tools {
		path, found, version := t.client.Check(ctx, current[t.key])
		results = append(results, CheckResult{Tool: t.name, Path: path, Found: found, Version: version})
	}

	path, found, version, backend, gpu := s.whisper.CheckWithBackend(ctx, current[KeyWhisperPath])
	results = append(results, CheckResult{Tool: "whisper", Path: path, Found: found, Version: version, Backend: backend, GPU: gpu})

	modelPath, modelFound, _ := s.whisperModel.CheckModel(ctx, current[KeyWhisperModel])
	results = append(results, CheckResult{Tool: "model whisper", Path: modelPath, Found: modelFound})

	return results, nil
}
