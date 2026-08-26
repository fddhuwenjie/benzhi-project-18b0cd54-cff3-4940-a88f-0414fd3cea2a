package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

const testDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const handoverDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const retestDigest = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
const contaminationDigest = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
const reviewDigest = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
const releaseDigest = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
const laterTestDigest = "1111111111111111111111111111111111111111111111111111111111111111"

func newTestService(t *testing.T) *Service {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	service.now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	return service
}

func createAndFreeze(t *testing.T, s *Service) *domain.SampleBatch {
	t.Helper()
	batch, err := s.CreateBatch(domain.CreateBatchCommand{WriteMeta: domain.WriteMeta{RequestID: "create"}, BatchID: "b1", SiteCode: "site", StratumReference: "L1", CollectorID: "collector", CollectedAt: s.now(), BaselineOxygenPPM: 20, BaselineTemperatureC: 8})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.FreezePlan(domain.FreezePlanCommand{WriteMeta: domain.WriteMeta{RequestID: "plan", ExpectedRevision: batch.Revision}, BatchID: "b1", HandoverAt: s.now(), HandoverEvidenceDigest: handoverDigest, Plan: domain.PreservationPlan{ContainerID: "jar", SealMethod: "double", CultureTarget: "target", CustodianID: "carrier", MaxOxygenPPM: 100, MinTemperatureC: 2, MaxTemperatureC: 12}})
	if err != nil {
		t.Fatal(err)
	}
	return batch
}

func TestDeviationMustPassRetest(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	batch, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp", ExpectedRevision: batch.Revision}, BatchID: "b1", RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 200, TemperatureC: 8, SealIntact: true, LocationNote: "gate", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != domain.StatusQuarantined || len(batch.Deviations) != 1 {
		t.Fatalf("未隔离: %+v", batch)
	}
	_, err = s.ResolveDeviation(domain.ResolveDeviationCommand{WriteMeta: domain.WriteMeta{RequestID: "resolve-bad", ExpectedRevision: batch.Revision}, BatchID: "b1", DeviationID: batch.Deviations[0].DeviationID, CorrectiveAction: "reseal", ResolvedBy: "operator", ResolutionDigest: testDigest, Retest: domain.AddCheckpointCommand{RecordedBy: "operator", RecordedAt: s.now(), OxygenPPM: 150, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest}})
	if err == nil {
		t.Fatal("超限复测不应闭环")
	}
	batch, err = s.ResolveDeviation(domain.ResolveDeviationCommand{WriteMeta: domain.WriteMeta{RequestID: "resolve-good", ExpectedRevision: batch.Revision}, BatchID: "b1", DeviationID: batch.Deviations[0].DeviationID, CorrectiveAction: "reseal", ResolvedBy: "operator", ResolutionDigest: releaseDigest, Retest: domain.AddCheckpointCommand{RecordedBy: "operator", RecordedAt: s.now().Add(time.Minute), OxygenPPM: 50, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: retestDigest}})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != domain.StatusAwaitingTest || batch.Deviations[0].ResolvedAt == nil {
		t.Fatal("合格复测未闭环")
	}
}

func TestReleaseArchivesAndIdempotencySurvivesRestart(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	var err error
	batch, err = s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp", ExpectedRevision: 2}, BatchID: "b1", RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "test", ExpectedRevision: batch.Revision}, BatchID: "b1", Result: "not_detected", TestedBy: "tester", TestedAt: s.now(), Method: "blank", EvidenceDigest: contaminationDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "review", ExpectedRevision: batch.Revision}, BatchID: "b1", ReviewerID: "reviewer", ReviewedAt: s.now(), Decision: "approve", EvidenceDigest: reviewDigest})
	if err != nil {
		t.Fatal(err)
	}
	cmd := domain.ReleaseCommand{WriteMeta: domain.WriteMeta{RequestID: "release", ExpectedRevision: batch.Revision}, BatchID: "b1", IssuerID: "issuer", EvidenceDigest: releaseDigest}
	cert, err := s.Release(cmd)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Release(cmd)
	if err != nil {
		t.Fatalf("幂等重试失败: %v", err)
	}
	if cert.CertificateID != again.CertificateID {
		t.Fatal("幂等结果不一致")
	}
	stored, err := s.Batch("b1")
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Status.Archived() {
		t.Fatal("未归档")
	}
	if _, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "late", ExpectedRevision: stored.Revision}, BatchID: "b1", RecordedBy: "x", RecordedAt: s.now(), OxygenPPM: 1, TemperatureC: 8, SealIntact: true, LocationNote: "late", EvidenceDigest: testDigest}); err == nil {
		t.Fatal("归档后仍可写入")
	}
	evidence, err := s.Evidence("b1")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Verification != "verified" || evidence.IntegrityDigest != cert.IntegrityDigest || len(evidence.EvidenceManifest) != 5 {
		t.Fatalf("归档核验结果不完整: %+v", evidence)
	}
	againEvidence, err := s.Evidence("b1")
	if err != nil || againEvidence.IntegrityDigest != evidence.IntegrityDigest {
		t.Fatalf("重复读取归档不稳定: %+v %v", againEvidence, err)
	}
	reopened, err := storage.Open(s.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	afterRestart := New(reopened)
	replayed, err := afterRestart.Release(cmd)
	if err != nil {
		t.Fatalf("跨重启幂等失败: %v", err)
	}
	if replayed.CertificateID != cert.CertificateID {
		t.Fatal("跨重启结果不一致")
	}
}

func TestFreezeValidatesBaselineHandoverAndIdempotency(t *testing.T) {
	s := newTestService(t)
	batch, err := s.CreateBatch(domain.CreateBatchCommand{WriteMeta: domain.WriteMeta{RequestID: "create-freeze"}, BatchID: "freeze-b1", SiteCode: "site", StratumReference: "L1", CollectorID: "collector", CollectedAt: s.now(), BaselineOxygenPPM: 200, BaselineTemperatureC: 8})
	if err != nil {
		t.Fatal(err)
	}
	head, _ := s.store.AuditHead()
	invalid := domain.FreezePlanCommand{WriteMeta: domain.WriteMeta{RequestID: "freeze-invalid", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, HandoverAt: s.now(), HandoverEvidenceDigest: handoverDigest, Plan: domain.PreservationPlan{ContainerID: "jar", SealMethod: "double", CultureTarget: "target", CustodianID: "carrier", MaxOxygenPPM: 100, MinTemperatureC: 2, MaxTemperatureC: 12}}
	if _, err := s.FreezePlan(invalid); errorCode(err) != "validation_error" {
		t.Fatalf("初始读数不适配应返回 validation_error: %v", err)
	}
	unchanged, _ := s.Batch(batch.BatchID)
	afterHead, _ := s.store.AuditHead()
	if unchanged.Status != domain.StatusDraft || unchanged.Revision != batch.Revision || afterHead != head {
		t.Fatalf("冻结校验失败后状态发生变化: %+v", unchanged)
	}
	valid := invalid
	valid.RequestID = "freeze-valid"
	valid.Plan.MaxOxygenPPM = 300
	frozen, err := s.FreezePlan(valid)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Status != domain.StatusReadyTransfer || frozen.HandoverConfirmation == nil || frozen.PreservationPlanSummary == "" {
		t.Fatalf("冻结结果缺少交接确认或方案摘要: %+v", frozen)
	}
	replayed, err := s.FreezePlan(valid)
	if err != nil || replayed.Revision != frozen.Revision {
		t.Fatalf("冻结幂等重放失败: %+v %v", replayed, err)
	}
	changed := valid
	changed.HandoverEvidenceDigest = testDigest
	if _, err := s.FreezePlan(changed); errorCode(err) != "idempotency_conflict" {
		t.Fatalf("变更交接摘要重试应冲突: %v", err)
	}
}

func TestCheckpointSequenceOrderingAndEvidenceUniqueness(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	first, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp-1", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, CheckpointID: "checkpoint-one", RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "gate", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp-2", ExpectedRevision: first.Revision}, BatchID: batch.BatchID, CheckpointID: "checkpoint-two", RecordedBy: "carrier", RecordedAt: s.now().Add(time.Minute), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: retestDigest})
	if err != nil {
		t.Fatal(err)
	}
	if second.Checkpoints[0].Sequence != 1 || !second.Checkpoints[0].SequenceStart || second.Checkpoints[1].Sequence != 2 || second.Checkpoints[1].PreviousCheckpointID != "checkpoint-one" || second.TransferContinuity.TotalCheckpoints != 2 || second.TransferContinuity.HasBreak {
		t.Fatalf("检查点连续序列错误: %+v %+v", second.Checkpoints, second.TransferContinuity)
	}
	revision := second.Revision
	head, _ := s.store.AuditHead()
	_, err = s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp-old", ExpectedRevision: revision}, BatchID: batch.BatchID, CheckpointID: "checkpoint-old", RecordedBy: "carrier", RecordedAt: s.now().Add(30 * time.Second), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "old", EvidenceDigest: contaminationDigest})
	if errorCode(err) != "validation_error" {
		t.Fatalf("乱序检查点应被拒绝: %v", err)
	}
	_, err = s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp-duplicate-evidence", ExpectedRevision: revision}, BatchID: batch.BatchID, CheckpointID: "checkpoint-three", RecordedBy: "carrier", RecordedAt: s.now().Add(2 * time.Minute), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: retestDigest})
	if errorCode(err) != "evidence_conflict" {
		t.Fatalf("重复现场证据应被拒绝: %v", err)
	}
	stored, _ := s.Batch(batch.BatchID)
	afterHead, _ := s.store.AuditHead()
	if stored.Revision != revision || len(stored.Checkpoints) != 2 || head != afterHead {
		t.Fatal("失败的检查点命令改变了聚合或审计头")
	}
}

func TestContaminationAndReviewHistoryKeepsVersions(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	batch, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "history-cp", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "detected", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, Result: "detected", TestedBy: "tester-one", TestedAt: s.now(), Method: "culture", EvidenceDigest: contaminationDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "not-detected", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, Result: "not_detected", TestedBy: "tester-two", TestedAt: s.now().Add(time.Minute), Method: "culture", EvidenceDigest: reviewDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "history-review", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, ReviewerID: "quality", ReviewedAt: s.now().Add(time.Minute), Decision: "approve", EvidenceDigest: releaseDigest})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.ContaminationHistory) != 2 || len(batch.ReviewHistory) != 1 || batch.CurrentReleaseVersion.ContaminationAttempt != 2 || batch.CurrentReleaseVersion.QualityReviewAttempt != 1 || batch.Review.ContaminationTestID != batch.Contamination.TestID {
		t.Fatalf("检测复核履历或当前版本错误: %+v", batch)
	}
	_, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "duplicate-review", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, ReviewerID: "quality-two", ReviewedAt: s.now().Add(2 * time.Minute), Decision: "approve", EvidenceDigest: releaseDigest})
	if errorCode(err) != "invalid_state" && errorCode(err) != "evidence_conflict" {
		t.Fatalf("陈旧状态或重复复核证据应被拒绝: %v", err)
	}
}

func TestRejectedReviewRequiresNewPairedVersion(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	batch, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "reject-cp", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "reject-test", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, Result: "not_detected", TestedBy: "tester", TestedAt: s.now(), Method: "culture", EvidenceDigest: contaminationDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "reject-review", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, ReviewerID: "quality", ReviewedAt: s.now(), Decision: "reject", EvidenceDigest: reviewDigest})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != domain.StatusAwaitingTest || len(batch.ReviewHistory) != 1 {
		t.Fatalf("驳回复核未被保留或状态错误: %+v", batch)
	}
	_, err = s.Release(domain.ReleaseCommand{WriteMeta: domain.WriteMeta{RequestID: "blocked-release", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, IssuerID: "issuer", EvidenceDigest: releaseDigest})
	if errorCode(err) != "release_blocked" {
		t.Fatalf("驳回后放行应被阻止: %v", err)
	}
	revision := batch.Revision
	_, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "reused-test-evidence", ExpectedRevision: revision}, BatchID: batch.BatchID, Result: "not_detected", TestedBy: "tester-two", TestedAt: s.now().Add(time.Minute), Method: "culture", EvidenceDigest: contaminationDigest})
	if errorCode(err) != "evidence_conflict" {
		t.Fatalf("历史检测证据不应被复用: %v", err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "new-test-version", ExpectedRevision: revision}, BatchID: batch.BatchID, Result: "not_detected", TestedBy: "tester-two", TestedAt: s.now().Add(time.Minute), Method: "culture", EvidenceDigest: laterTestDigest})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "same-person-review", ExpectedRevision: batch.Revision}, BatchID: batch.BatchID, ReviewerID: "tester-two", ReviewedAt: s.now().Add(time.Minute), Decision: "approve", EvidenceDigest: releaseDigest})
	if errorCode(err) != "validation_error" {
		t.Fatalf("复核员与最新检测员相同时应被拒绝: %v", err)
	}
}

func errorCode(err error) string {
	var target *domain.Error
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

func TestRevisionAndRoleSeparation(t *testing.T) {
	s := newTestService(t)
	batch := createAndFreeze(t, s)
	_, err := s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "stale", ExpectedRevision: 1}, BatchID: "b1", RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest})
	if err == nil {
		t.Fatal("陈旧版本应失败")
	}
	batch, err = s.AddCheckpoint(domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: "cp", ExpectedRevision: batch.Revision}, BatchID: "b1", RecordedBy: "carrier", RecordedAt: s.now(), OxygenPPM: 20, TemperatureC: 8, SealIntact: true, LocationNote: "lab", EvidenceDigest: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = s.RecordContamination(domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: "test", ExpectedRevision: batch.Revision}, BatchID: "b1", Result: "not_detected", TestedBy: "tester", TestedAt: s.now(), Method: "blank", EvidenceDigest: contaminationDigest})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Review(domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: "review", ExpectedRevision: batch.Revision}, BatchID: "b1", ReviewerID: "collector", ReviewedAt: s.now(), Decision: "approve", EvidenceDigest: testDigest})
	if err == nil {
		t.Fatal("采样员不得复核")
	}
}
