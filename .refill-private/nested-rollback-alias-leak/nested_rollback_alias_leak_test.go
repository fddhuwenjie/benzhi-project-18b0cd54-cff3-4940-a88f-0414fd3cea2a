package nested_rollback_alias_leak_test

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestFailedTransactionDoesNotPersistNestedMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	store, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recordedAt := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	planFrozenAt := recordedAt
	err = store.Update(func(state *storage.State) error {
		contamination := domain.ContaminationTest{TestID: "test-one", Attempt: 1, RecordedRevision: 1, Result: "not_detected", TestedAt: recordedAt, Method: "culture-before", EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
		review := domain.QualityReview{ReviewID: "review-one", Attempt: 1, ContaminationTestID: contamination.TestID, RecordedRevision: 1, ReviewerID: "reviewer-before", ReviewedAt: recordedAt, Decision: "approve", EvidenceDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
		state.Batches["rollback-batch"] = &domain.SampleBatch{
			BatchID:              "rollback-batch",
			Status:               domain.StatusDraft,
			Revision:             1,
			PreservationPlan:     &domain.PreservationPlan{ContainerID: "container-before"},
			HandoverConfirmation: &domain.HandoverConfirmation{CustodianID: "custodian-before"},
			PlanFrozenAt:         &planFrozenAt,
			Checkpoints: []domain.TransferCheckpoint{{
				Sequence:       1,
				SequenceStart:  true,
				CheckpointID:   "checkpoint-one",
				BatchID:        "rollback-batch",
				RecordedAt:     recordedAt,
				LocationNote:   "sealed transport box",
				EvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				WithinLimits:   true,
			}},
			TransferContinuity: domain.TransferContinuitySummary{
				LatestRecordedAt: &recordedAt,
				TotalCheckpoints: 1,
			},
			Deviations:           []domain.DeviationCase{{DeviationID: "deviation-one", BatchID: "rollback-batch", CheckpointID: "checkpoint-one", ContainmentAction: "contain-before"}},
			Contamination:        &contamination,
			ContaminationHistory: []domain.ContaminationTest{contamination},
			Review:               &review,
			ReviewHistory:        []domain.QualityReview{review},
			CurrentReleaseVersion: domain.CurrentReleaseVersion{
				ContaminationTestID:  contamination.TestID,
				ContaminationAttempt: 1,
				QualityReviewID:      review.ReviewID,
				QualityReviewAttempt: 1,
			},
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	rollback := errors.New("force transaction rollback")
	err = store.Update(func(state *storage.State) error {
		batch := state.Batches["rollback-batch"]
		batch.PreservationPlan.ContainerID = "container-uncommitted"
		batch.HandoverConfirmation.CustodianID = "custodian-uncommitted"
		*batch.PlanFrozenAt = recordedAt.Add(time.Hour)
		batch.Checkpoints[0].LocationNote = "uncommitted quarantine shelf"
		batch.Deviations[0].ContainmentAction = "contain-uncommitted"
		batch.Contamination.Method = "culture-uncommitted"
		batch.ContaminationHistory[0].Method = "culture-uncommitted"
		batch.Review.ReviewerID = "reviewer-uncommitted"
		batch.ReviewHistory[0].ReviewerID = "reviewer-uncommitted"
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error lost: %v", err)
	}

	if err := store.Update(func(*storage.State) error { return nil }); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := reopened.Batch("rollback-batch")
	if err != nil {
		t.Fatal(err)
	}
	if batch.PreservationPlan.ContainerID != "container-before" ||
		batch.HandoverConfirmation.CustodianID != "custodian-before" ||
		!batch.PlanFrozenAt.Equal(recordedAt) ||
		batch.Checkpoints[0].LocationNote != "sealed transport box" ||
		batch.Deviations[0].ContainmentAction != "contain-before" ||
		batch.Contamination.Method != "culture-before" ||
		batch.ContaminationHistory[0].Method != "culture-before" ||
		batch.Review.ReviewerID != "reviewer-before" ||
		batch.ReviewHistory[0].ReviewerID != "reviewer-before" {
		t.Fatalf("failed transaction mutation survived restart: batch=%+v", batch)
	}
}
