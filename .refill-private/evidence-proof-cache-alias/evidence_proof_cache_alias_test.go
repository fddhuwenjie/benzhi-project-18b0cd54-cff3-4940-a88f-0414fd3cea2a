package evidence_proof_cache_alias_test

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"path/filepath"
	"testing"
	"time"
)

const (
	planDigest    = "1111111111111111111111111111111111111111111111111111111111111111"
	checkpointDig = "2222222222222222222222222222222222222222222222222222222222222222"
	testDigest    = "3333333333333333333333333333333333333333333333333333333333333333"
	reviewDigest  = "4444444444444444444444444444444444444444444444444444444444444444"
	releaseDigest = "5555555555555555555555555555555555555555555555555555555555555555"
)

func TestEvidenceProofCacheOwnsReturnedSlice(t *testing.T) {
	service := archivedService(t)

	first, err := service.Evidence("proof-cache-batch")
	if err != nil {
		t.Fatalf("first evidence query: %v", err)
	}
	if len(first.Package.AuditProof) == 0 {
		t.Fatal("first evidence query returned no audit proof")
	}
	first.Package.AuditProof[0].Type = "caller.tampered"

	second, err := service.Evidence("proof-cache-batch")
	if err != nil {
		t.Fatalf("cached audit proof was poisoned by caller mutation: %v", err)
	}
	if second.Verification != "verified" {
		t.Fatalf("second evidence verification=%q", second.Verification)
	}
}

func archivedService(t *testing.T) *workflow.Service {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(store)
	base := time.Now().UTC()
	batch, err := service.CreateBatch(domain.CreateBatchCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-create"}, BatchID: "proof-cache-batch",
		SiteCode: "site-proof", StratumReference: "layer-proof", CollectorID: "collector-proof",
		CollectedAt: base, BaselineOxygenPPM: 20, BaselineTemperatureC: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.FreezePlan(domain.FreezePlanCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-plan", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID,
		HandoverAt: base, HandoverEvidenceDigest: planDigest,
		Plan: domain.PreservationPlan{ContainerID: "jar-proof", SealMethod: "double-seal", CultureTarget: "target-proof", CustodianID: "carrier-proof", MaxOxygenPPM: 100, MinTemperatureC: 2, MaxTemperatureC: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.AddCheckpoint(domain.AddCheckpointCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-checkpoint", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID,
		RecordedBy: "carrier-proof", RecordedAt: base.Add(time.Minute), OxygenPPM: 30, TemperatureC: 8,
		SealIntact: true, LocationNote: "lab handover", EvidenceDigest: checkpointDig,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.RecordContamination(domain.RecordContaminationCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-test", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID,
		Result: "not_detected", TestedBy: "tester-proof", TestedAt: base.Add(2 * time.Minute), Method: "blank-control", EvidenceDigest: testDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.Review(domain.ReviewCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-review", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID,
		ReviewerID: "reviewer-proof", ReviewedAt: base.Add(3 * time.Minute), Decision: "approve", EvidenceDigest: reviewDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Release(domain.ReleaseCommand{
		WriteMeta: domain.WriteMeta{RequestID: "proof-release", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID,
		IssuerID: "issuer-proof", EvidenceDigest: releaseDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
