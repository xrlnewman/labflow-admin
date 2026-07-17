package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSampleCreateRequiresAliasAndTests(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	ctx := context.Background()
	if _, err := svc.CreateSample(ctx, CreateSampleInput{SampleType: "血液", Tests: []string{"血常规"}}, "sample-missing-alias"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("missing alias error = %v", err)
	}
	if _, err := svc.CreateSample(ctx, CreateSampleInput{SubjectAlias: "样本-001", SampleType: "血液"}, "sample-empty-tests"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty tests error = %v", err)
	}
}

func TestSampleReportLifecycleRejectsEarlyReviewAndArchivesReport(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	ctx := context.Background()
	sample, err := svc.CreateSample(ctx, CreateSampleInput{SubjectAlias: "样本-002", SampleType: "血液", CollectedAt: "2026-07-17T09:00:00Z", Tests: []string{"血常规", "C反应蛋白"}}, "sample-002")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewSample(ctx, sample.ID, "审核员", "review-too-early"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("early review error = %v", err)
	}
	for _, step := range []struct {
		name string
		fn   func() (Sample, error)
	}{
		{"receive", func() (Sample, error) { return svc.ReceiveSample(ctx, sample.ID, "收样员", "receive-002") }},
		{"start-test", func() (Sample, error) { return svc.StartSampleTest(ctx, sample.ID, "检验员", "start-002") }},
	} {
		sample, err = step.fn()
		if err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}
	if _, err := svc.ReportSample(ctx, sample.ID, CreateReportInput{Result: "阴性", Remark: "结果稳定"}, "report-002"); err != nil {
		t.Fatal(err)
	}
	if sample, err = svc.ReviewSample(ctx, sample.ID, "审核员", "review-002"); err != nil || sample.Status != SampleStatusReported {
		t.Fatalf("review sample = %+v, err=%v", sample, err)
	}
	if sample.Report == nil || sample.Report.Status != SampleStatusReported {
		t.Fatalf("report after review = %+v", sample.Report)
	}
	if sample, err = svc.ArchiveSample(ctx, sample.ID, "归档员", "archive-002"); err != nil || sample.Status != SampleStatusArchived {
		t.Fatalf("archive sample = %+v, err=%v", sample, err)
	}
	if sample.Report == nil || sample.Report.Status != SampleStatusArchived {
		t.Fatalf("report after archive = %+v", sample.Report)
	}
	events, err := store.ListSampleEvents(ctx, sample.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(events), 5; got != want {
		t.Fatalf("events = %d, want %d", got, want)
	}
	for i := 1; i < len(events); i++ {
		if events[i-1].CreatedAt > events[i].CreatedAt {
			t.Fatalf("events out of order: %#v", events)
		}
	}
}

func TestSampleWritesAreIdempotent(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	input := CreateSampleInput{SubjectAlias: "样本-003", SampleType: "尿液", Tests: []string{"尿常规"}}
	a, err := svc.CreateSample(context.Background(), input, "same-sample-key")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateSample(context.Background(), input, "same-sample-key")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("idempotency returned %q then %q", a.ID, b.ID)
	}
	events, err := store.ListSampleEvents(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("create should not write transition events, got %d", len(events))
	}
}

func TestSampleActorIsRequired(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	sample, err := svc.CreateSample(context.Background(), CreateSampleInput{SubjectAlias: "样本-004", SampleType: "血液", Tests: []string{"血常规"}}, "sample-004")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReceiveSample(context.Background(), sample.ID, "", "receive-004"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty actor error = %v", err)
	}
}

func TestSampleTestsSchemaReferencesSamples(t *testing.T) {
	contents, err := os.ReadFile("../deploy/mysql/init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "FOREIGN KEY (sample_id) REFERENCES samples(id)") {
		t.Fatal("sample_tests must reference samples(id)")
	}
}
