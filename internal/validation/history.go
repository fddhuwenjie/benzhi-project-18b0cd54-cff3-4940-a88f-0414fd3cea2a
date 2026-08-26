package validation

import (
	"anaerobic-release/internal/domain"
	"time"
)

func CheckpointContinuity(batch *domain.SampleBatch, checkpointID, evidenceDigest string, recordedAt time.Time) error {
	for _, checkpoint := range batch.Checkpoints {
		if checkpoint.CheckpointID == checkpointID {
			return domain.NewError("checkpoint_conflict", "checkpoint_id 已在本批次使用", 409)
		}
		if checkpoint.EvidenceDigest == evidenceDigest {
			return domain.NewError("evidence_conflict", "evidence_digest 已用于本批次其他检查点", 409)
		}
	}
	if batch.PlanFrozenAt == nil {
		return state("批次缺少方案冻结时间")
	}
	if recordedAt.Before(*batch.PlanFrozenAt) {
		return invalid("recorded_at 不得早于 plan_frozen_at")
	}
	if len(batch.Checkpoints) > 0 && !recordedAt.After(batch.Checkpoints[len(batch.Checkpoints)-1].RecordedAt) {
		return invalid("recorded_at 必须严格晚于最近检查点")
	}
	return nil
}

func UniqueContaminationEvidence(history []domain.ContaminationTest, digest string) error {
	for _, attempt := range history {
		if attempt.EvidenceDigest == digest {
			return domain.NewError("evidence_conflict", "evidence_digest 已用于历史污染检测", 409)
		}
	}
	return nil
}

func UniqueReviewEvidence(history []domain.QualityReview, digest string) error {
	for _, attempt := range history {
		if attempt.EvidenceDigest == digest {
			return domain.NewError("evidence_conflict", "evidence_digest 已用于历史质量复核", 409)
		}
	}
	return nil
}
