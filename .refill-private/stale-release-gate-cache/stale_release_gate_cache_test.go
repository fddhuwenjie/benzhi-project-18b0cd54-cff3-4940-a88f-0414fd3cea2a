package stale_release_gate_cache_test

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"path/filepath"
	"testing"
	"time"
)

const (
	checkpointDigest    = "1111111111111111111111111111111111111111111111111111111111111111"
	handoverDigest      = "2222222222222222222222222222222222222222222222222222222222222222"
	contaminationDigest = "3333333333333333333333333333333333333333333333333333333333333333"
	reviewDigest        = "4444444444444444444444444444444444444444444444444444444444444444"
	releaseDigest       = "5555555555555555555555555555555555555555555555555555555555555555"
)

func TestReleaseGateRechecksAfterStateAdvance(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(store)
	base := time.Now().UTC().Add(-time.Hour)

	batch, err := service.CreateBatch(domain.CreateBatchCommand{
		WriteMeta: domain.WriteMeta{RequestID: "create-release-cache"},
		BatchID:   "release-cache-batch", SiteCode: "site-alpha", StratumReference: "layer-1",
		CollectorID: "collector-alpha", CollectedAt: base,
		BaselineOxygenPPM: 20, BaselineTemperatureC: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.FreezePlan(domain.FreezePlanCommand{
		WriteMeta: domain.WriteMeta{RequestID: "freeze-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, HandoverAt: base.Add(time.Minute), HandoverEvidenceDigest: handoverDigest,
		Plan: domain.PreservationPlan{ContainerID: "container-alpha", SealMethod: "double-seal", CultureTarget: "anaerobic-culture", CustodianID: "custodian-alpha", MaxOxygenPPM: 100, MinTemperatureC: 2, MaxTemperatureC: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	lifecycleAt := *batch.PlanFrozenAt
	batch, err = service.AddCheckpoint(domain.AddCheckpointCommand{
		WriteMeta: domain.WriteMeta{RequestID: "checkpoint-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, CheckpointID: "checkpoint-alpha", RecordedBy: "carrier-alpha",
		RecordedAt: lifecycleAt, OxygenPPM: 25, TemperatureC: 8, SealIntact: true,
		LocationNote: "laboratory-arrival", EvidenceDigest: checkpointDigest,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Release(domain.ReleaseCommand{
		WriteMeta: domain.WriteMeta{RequestID: "early-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, IssuerID: "issuer-alpha", EvidenceDigest: releaseDigest,
	})
	if err == nil {
		t.Fatal("early release unexpectedly succeeded")
	}

	batch, err = service.RecordContamination(domain.RecordContaminationCommand{
		WriteMeta: domain.WriteMeta{RequestID: "test-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, Result: "not_detected", TestedBy: "tester-alpha",
		TestedAt: lifecycleAt, Method: "culture-blank", EvidenceDigest: contaminationDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.Review(domain.ReviewCommand{
		WriteMeta: domain.WriteMeta{RequestID: "review-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, ReviewerID: "reviewer-alpha", ReviewedAt: lifecycleAt,
		Decision: "approve", EvidenceDigest: reviewDigest,
	})
	if err != nil {
		t.Fatal(err)
	}

	certificate, err := service.Release(domain.ReleaseCommand{
		WriteMeta: domain.WriteMeta{RequestID: "final-release-cache", ExpectedRevision: batch.Revision},
		BatchID:   batch.BatchID, IssuerID: "issuer-alpha", EvidenceDigest: releaseDigest,
	})
	if err != nil {
		t.Fatalf("release gate reused stale decision after revision advanced: %v", err)
	}
	if certificate == nil || certificate.BatchID != batch.BatchID {
		t.Fatalf("release returned an invalid certificate: %+v", certificate)
	}
}
