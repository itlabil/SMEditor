package project

import (
	"context"
	"fmt"
	"os"

	"smeditor/internal/modules/job"
	"smeditor/internal/modules/settings"
	"smeditor/internal/storage"
	"smeditor/internal/tools/whisper"
)

// AudioExtractor pulls a 16kHz mono WAV out of a video file. Implemented
// by tools/ffmpeg.Ffmpeg.
type AudioExtractor interface {
	ExtractAudio(ctx context.Context, ffmpegPath, videoPath, outPath string) error
}

// WhisperClient runs the local Whisper model. Implemented by
// tools/whisper.Client.
type WhisperClient interface {
	Transcribe(ctx context.Context, whisperPath string, opts whisper.TranscribeOptions, report whisper.ProgressFunc) error
}

const whisperRawOutputBase = "transcript_raw"

// TranscribeRunner implements job.Runner for job.TypeTranscribe: it
// extracts audio, runs Whisper, and writes transcript.json/transcript.txt,
// per docs/flow.md section 4.3.
type TranscribeRunner struct {
	repo     *Repository
	storage  *storage.Storage
	settings SettingsReader
	audio    AudioExtractor
	whisper  WhisperClient
}

func NewTranscribeRunner(
	repo *Repository,
	st *storage.Storage,
	settingsReader SettingsReader,
	audioExtractor AudioExtractor,
	whisperClient WhisperClient,
) *TranscribeRunner {
	return &TranscribeRunner{repo: repo, storage: st, settings: settingsReader, audio: audioExtractor, whisper: whisperClient}
}

func (r *TranscribeRunner) Type() string { return job.TypeTranscribe }

func (r *TranscribeRunner) Run(ctx context.Context, j job.Job, report job.ProgressFunc) error {
	p, err := r.repo.FindByID(ctx, j.ProjectID)
	if err != nil {
		return fmt.Errorf("baca project: %w", err)
	}

	cfg, err := r.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("baca pengaturan: %w", err)
	}

	videoPath, err := r.storage.FilePath(j.ProjectID, storage.SourceVideoFile)
	if err != nil {
		return err
	}
	audioPath, err := r.storage.FilePath(j.ProjectID, storage.AudioFile)
	if err != nil {
		return err
	}
	if err := r.audio.ExtractAudio(ctx, cfg[settings.KeyFfmpegPath], videoPath, audioPath); err != nil {
		return fmt.Errorf("ekstrak audio: %w", err)
	}

	outBase, err := r.storage.FilePath(j.ProjectID, whisperRawOutputBase)
	if err != nil {
		return err
	}
	defer os.Remove(outBase + ".json")

	lang := p.TranscriptLang
	if lang == "" {
		lang = "auto"
	}

	err = r.whisper.Transcribe(ctx, cfg[settings.KeyWhisperPath], whisper.TranscribeOptions{
		AudioPath:  audioPath,
		OutputBase: outBase,
		ModelPath:  cfg[settings.KeyWhisperModel],
		Language:   lang,
		NoGPU:      cfg[settings.KeyWhisperDevice] == "cpu",
	}, func(percent float64, message string) {
		report(percent, message)
	})
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(outBase + ".json")
	if err != nil {
		return fmt.Errorf("baca hasil whisper: %w", err)
	}
	transcript, err := whisper.ParseRawOutput(raw, p.DurationSec)
	if err != nil {
		return err
	}

	jsonBytes, err := transcript.JSON()
	if err != nil {
		return fmt.Errorf("susun transcript.json: %w", err)
	}

	jsonPath, err := r.storage.FilePath(j.ProjectID, storage.TranscriptJSONFile)
	if err != nil {
		return err
	}
	txtPath, err := r.storage.FilePath(j.ProjectID, storage.TranscriptTXTFile)
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("simpan transcript.json: %w", err)
	}
	if err := os.WriteFile(txtPath, []byte(transcript.TXT()), 0o644); err != nil {
		return fmt.Errorf("simpan transcript.txt: %w", err)
	}

	return nil
}
