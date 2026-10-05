package project

import (
	"context"
	"testing"

	"smeditor/internal/modules/job"
)

func TestJobSync_OnJobStarted_SetsStatusPerType(t *testing.T) {
	cases := []struct {
		jobType    string
		wantStatus string
		wantNoop   bool
	}{
		{job.TypeDownload, StatusMengunduh, false},
		{job.TypeConvert, StatusMengunduh, false},
		{job.TypeTranscribe, StatusTranscript, false},
	}
	for _, tc := range cases {
		t.Run(tc.jobType, func(t *testing.T) {
			repo, st := newTestRepoAndStorage(t)
			p := seedTestProject(t, repo, st)
			sync := NewJobSync(repo, &fakeEnqueuer{})

			if err := sync.OnJobStarted(context.Background(), p.ID, tc.jobType); err != nil {
				t.Fatalf("OnJobStarted: %v", err)
			}

			got, err := repo.FindByID(context.Background(), p.ID)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tc.wantStatus)
			}
		})
	}
}

func TestJobSync_OnJobFinished_Failed_SetsFailureStatusAndMessage(t *testing.T) {
	cases := []struct {
		jobType    string
		wantStatus string
	}{
		{job.TypeDownload, StatusGagalUnduh},
		{job.TypeConvert, StatusGagalUnduh},
		{job.TypeTranscribe, StatusGagalTranscript},
	}
	for _, tc := range cases {
		t.Run(tc.jobType, func(t *testing.T) {
			repo, st := newTestRepoAndStorage(t)
			p := seedTestProject(t, repo, st)
			sync := NewJobSync(repo, &fakeEnqueuer{})

			if err := sync.OnJobFinished(context.Background(), p.ID, tc.jobType, job.StatusFailed, "pesan error dari tool"); err != nil {
				t.Fatalf("OnJobFinished: %v", err)
			}

			got, err := repo.FindByID(context.Background(), p.ID)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tc.wantStatus)
			}
			if got.ErrorMessage != "pesan error dari tool" {
				t.Errorf("error_message = %q, want the failure message", got.ErrorMessage)
			}
		})
	}
}

func TestJobSync_OnJobFinished_ConvertDoneEnqueuesTranscribe(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	jobs := &fakeEnqueuer{}
	sync := NewJobSync(repo, jobs)

	if err := sync.OnJobFinished(context.Background(), p.ID, job.TypeConvert, job.StatusDone, ""); err != nil {
		t.Fatalf("OnJobFinished: %v", err)
	}

	if len(jobs.calls) != 1 || jobs.calls[0] != job.TypeTranscribe {
		t.Errorf("enqueued = %v, want exactly [transcribe]", jobs.calls)
	}
}

func TestJobSync_OnJobFinished_TranscribeDoneSetsMenungguHighlight(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	sync := NewJobSync(repo, &fakeEnqueuer{})

	if err := sync.OnJobFinished(context.Background(), p.ID, job.TypeTranscribe, job.StatusDone, ""); err != nil {
		t.Fatalf("OnJobFinished: %v", err)
	}

	got, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusMenungguHighlight {
		t.Errorf("status = %q, want %q", got.Status, StatusMenungguHighlight)
	}
}

func TestJobSync_OnJobFinished_DownloadDoneIsNoop(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	jobs := &fakeEnqueuer{}
	sync := NewJobSync(repo, jobs)

	before, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if err := sync.OnJobFinished(context.Background(), p.ID, job.TypeDownload, job.StatusDone, ""); err != nil {
		t.Fatalf("OnJobFinished: %v", err)
	}

	after, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if after.Status != before.Status {
		t.Errorf("status changed from %q to %q; download-done must be a no-op (DownloadRunner itself enqueues next)", before.Status, after.Status)
	}
	if len(jobs.calls) != 0 {
		t.Errorf("enqueued = %v, want none", jobs.calls)
	}
}

func TestJobSync_OnJobFinished_CanceledLeavesStatusUntouched(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	sync := NewJobSync(repo, &fakeEnqueuer{})

	if err := sync.OnJobStarted(context.Background(), p.ID, job.TypeDownload); err != nil {
		t.Fatalf("OnJobStarted: %v", err)
	}
	if err := sync.OnJobFinished(context.Background(), p.ID, job.TypeDownload, job.StatusCanceled, ""); err != nil {
		t.Fatalf("OnJobFinished: %v", err)
	}

	got, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusMengunduh {
		t.Errorf("status = %q, want it to stay %q after a cancel", got.Status, StatusMengunduh)
	}
}
