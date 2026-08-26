package storage

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
	"fmt"
	"reflect"
	"time"
)

func prepareRecovery(state *State) error {
	state.Recovery.Generation++
	state.Recovery.SavedAt = time.Now().UTC()
	state.Recovery.BatchCount = len(state.Batches)
	state.Recovery.ArchivedCount = len(state.ArchiveIndex)
	state.Recovery.AuditHead = state.Audit.Head()
	digest, err := contentDigest(state)
	if err != nil {
		return err
	}
	state.Recovery.ContentDigest = digest
	return nil
}

func verifyState(state *State, checkRecovery bool) error {
	if state.SchemaVersion != 1 {
		return fmt.Errorf("不支持的 schema_version=%d", state.SchemaVersion)
	}
	if err := state.Audit.Verify(); err != nil {
		return fmt.Errorf("审计链无效: %w", err)
	}
	for key, batch := range state.Batches {
		if batch == nil {
			return fmt.Errorf("批次 %s 聚合为空", key)
		}
		if key != batch.BatchID {
			return fmt.Errorf("批次索引 %s 与实体 ID %s 不一致", key, batch.BatchID)
		}
		if err := verifyBatch(batch); err != nil {
			return fmt.Errorf("批次 %s: %w", key, err)
		}
	}
	for batchID, record := range state.ArchiveIndex {
		batch, exists := state.Batches[batchID]
		if !exists {
			return fmt.Errorf("归档索引引用不存在的批次 %s", batchID)
		}
		if err := verifyArchive(batch, record, state.Audit); err != nil {
			return fmt.Errorf("归档 %s: %w", batchID, err)
		}
	}
	if checkRecovery && state.Recovery.Generation > 0 {
		if err := verifyRecovery(state); err != nil {
			return err
		}
	}
	return nil
}

func verifyBatch(batch *domain.SampleBatch) error {
	if batch.BatchID == "" || batch.Revision < 1 {
		return fmt.Errorf("标识或 revision 无效")
	}
	if batch.Status != domain.StatusDraft {
		if batch.PreservationPlan == nil || batch.HandoverConfirmation == nil || batch.PlanFrozenAt == nil || batch.PreservationPlanSummary == "" {
			return fmt.Errorf("非草稿批次缺少方案冻结交接信息")
		}
		summary, err := audit.Digest(*batch.PreservationPlan)
		if err != nil || summary != batch.PreservationPlanSummary {
			return fmt.Errorf("保存方案摘要不一致")
		}
	}
	checkpointIDs := make(map[string]struct{}, len(batch.Checkpoints))
	checkpointDigests := make(map[string]struct{}, len(batch.Checkpoints))
	expectedContinuity := domain.TransferContinuitySummary{TotalCheckpoints: len(batch.Checkpoints)}
	for index, checkpoint := range batch.Checkpoints {
		if checkpoint.BatchID != batch.BatchID || checkpoint.CheckpointID == "" {
			return fmt.Errorf("检查点归属或标识无效")
		}
		if checkpoint.Sequence != int64(index+1) || checkpoint.SequenceStart != (index == 0) {
			return fmt.Errorf("检查点 %s 序号或起点标记不连续", checkpoint.CheckpointID)
		}
		if index > 0 {
			previous := batch.Checkpoints[index-1]
			if checkpoint.PreviousCheckpointID != previous.CheckpointID || !checkpoint.RecordedAt.After(previous.RecordedAt) {
				return fmt.Errorf("检查点 %s 前序标识或时间不连续", checkpoint.CheckpointID)
			}
		} else if checkpoint.PreviousCheckpointID != "" {
			return fmt.Errorf("首个检查点包含前序标识")
		}
		if _, duplicate := checkpointIDs[checkpoint.CheckpointID]; duplicate {
			return fmt.Errorf("检查点 %s 重复", checkpoint.CheckpointID)
		}
		checkpointIDs[checkpoint.CheckpointID] = struct{}{}
		if _, duplicate := checkpointDigests[checkpoint.EvidenceDigest]; duplicate {
			return fmt.Errorf("检查点 %s 重复使用证据摘要", checkpoint.CheckpointID)
		}
		checkpointDigests[checkpoint.EvidenceDigest] = struct{}{}
		if !checkpoint.WithinLimits {
			expectedContinuity.HasLimitBreach = true
		}
	}
	if len(batch.Checkpoints) > 0 {
		latest := batch.Checkpoints[len(batch.Checkpoints)-1].RecordedAt
		expectedContinuity.LatestRecordedAt = &latest
	}
	if !reflect.DeepEqual(expectedContinuity, batch.TransferContinuity) {
		return fmt.Errorf("转运连续性摘要不一致")
	}
	deviationIDs := make(map[string]struct{}, len(batch.Deviations))
	for _, deviation := range batch.Deviations {
		if deviation.BatchID != batch.BatchID || deviation.DeviationID == "" {
			return fmt.Errorf("偏差归属或标识无效")
		}
		if _, exists := checkpointIDs[deviation.CheckpointID]; !exists {
			return fmt.Errorf("偏差 %s 引用不存在的检查点", deviation.DeviationID)
		}
		if _, duplicate := deviationIDs[deviation.DeviationID]; duplicate {
			return fmt.Errorf("偏差 %s 重复", deviation.DeviationID)
		}
		deviationIDs[deviation.DeviationID] = struct{}{}
		if deviation.ResolvedAt != nil {
			if _, exists := checkpointIDs[deviation.RetestCheckpointID]; !exists {
				return fmt.Errorf("偏差 %s 缺少有效复测检查点", deviation.DeviationID)
			}
		}
	}
	contaminationDigests := map[string]struct{}{}
	for index, attempt := range batch.ContaminationHistory {
		if attempt.Attempt != index+1 || attempt.TestID == "" || attempt.RecordedRevision < 1 {
			return fmt.Errorf("污染检测尝试 %d 版本字段无效", index+1)
		}
		if index > 0 && !attempt.TestedAt.After(batch.ContaminationHistory[index-1].TestedAt) {
			return fmt.Errorf("污染检测尝试时间不递增")
		}
		if _, duplicate := contaminationDigests[attempt.EvidenceDigest]; duplicate {
			return fmt.Errorf("污染检测证据摘要重复")
		}
		contaminationDigests[attempt.EvidenceDigest] = struct{}{}
	}
	if batch.Contamination != nil {
		if len(batch.ContaminationHistory) == 0 || *batch.Contamination != batch.ContaminationHistory[len(batch.ContaminationHistory)-1] {
			return fmt.Errorf("当前污染检测未指向最新尝试")
		}
		if batch.CurrentReleaseVersion.ContaminationTestID != batch.Contamination.TestID || batch.CurrentReleaseVersion.ContaminationAttempt != batch.Contamination.Attempt {
			return fmt.Errorf("当前放行版本未指向最新污染检测")
		}
	}
	reviewDigests := map[string]struct{}{}
	for index, attempt := range batch.ReviewHistory {
		if attempt.Attempt != index+1 || attempt.ReviewID == "" || attempt.ContaminationTestID == "" || attempt.RecordedRevision < 1 {
			return fmt.Errorf("质量复核尝试 %d 版本字段无效", index+1)
		}
		if index > 0 && !attempt.ReviewedAt.After(batch.ReviewHistory[index-1].ReviewedAt) {
			return fmt.Errorf("质量复核尝试时间不递增")
		}
		if _, duplicate := reviewDigests[attempt.EvidenceDigest]; duplicate {
			return fmt.Errorf("质量复核证据摘要重复")
		}
		reviewDigests[attempt.EvidenceDigest] = struct{}{}
	}
	if batch.Review != nil {
		if len(batch.ReviewHistory) == 0 || *batch.Review != batch.ReviewHistory[len(batch.ReviewHistory)-1] || batch.Contamination == nil || batch.Review.ContaminationTestID != batch.Contamination.TestID {
			return fmt.Errorf("当前质量复核未与最新检测配对")
		}
		if batch.CurrentReleaseVersion.QualityReviewID != batch.Review.ReviewID || batch.CurrentReleaseVersion.QualityReviewAttempt != batch.Review.Attempt {
			return fmt.Errorf("当前放行版本未指向最新质量复核")
		}
	} else if batch.CurrentReleaseVersion.QualityReviewID != "" || batch.CurrentReleaseVersion.QualityReviewAttempt != 0 {
		return fmt.Errorf("当前放行版本引用不存在的质量复核")
	}
	if batch.Status.Archived() {
		if batch.ArchivedAt == nil || batch.Certificate == nil {
			return fmt.Errorf("已归档批次缺少时间或证书")
		}
	} else if batch.Certificate != nil || batch.ArchivedAt != nil {
		return fmt.Errorf("未归档批次包含归档字段")
	}
	return nil
}

func verifyArchive(batch *domain.SampleBatch, record ArchiveRecord, chain audit.Chain) error {
	if !batch.Status.Archived() || batch.Certificate == nil {
		return fmt.Errorf("批次并非只读归档状态")
	}
	if record.BatchID != batch.BatchID || record.CertificateID != batch.Certificate.CertificateID {
		return fmt.Errorf("证书索引不一致")
	}
	if record.Revision != batch.Revision || record.AuditHead != batch.Certificate.AuditHead {
		return fmt.Errorf("revision 或 audit_head 不一致")
	}
	if record.IntegrityDigest != batch.Certificate.IntegrityDigest || record.ManifestItems != len(batch.Certificate.EvidenceManifest) {
		return fmt.Errorf("归档完整性摘要或清单计数不一致")
	}
	if err := audit.VerifyManifest(chain, batch.BatchID, batch.Certificate.EvidenceManifest); err != nil {
		return fmt.Errorf("证据清单无效: %w", err)
	}
	if record.ArchivedAt.IsZero() || !record.ArchivedAt.Equal(*batch.ArchivedAt) {
		return fmt.Errorf("归档时间不一致")
	}
	if !chain.Contains(record.AuditHead) {
		return fmt.Errorf("audit_head 不存在于审计链")
	}
	releaseEvent, err := chain.ReleaseHead(batch.BatchID, batch.Revision)
	if err != nil {
		return err
	}
	if releaseEvent.Hash != record.AuditHead {
		return fmt.Errorf("audit_head 不是批次对应的放行事件")
	}
	return nil
}

func verifyRecovery(state *State) error {
	metadata := state.Recovery
	if metadata.SavedAt.IsZero() || metadata.Generation < 1 {
		return fmt.Errorf("恢复元数据时间或代次无效")
	}
	if metadata.BatchCount != len(state.Batches) || metadata.ArchivedCount != len(state.ArchiveIndex) {
		return fmt.Errorf("恢复元数据计数不一致")
	}
	if metadata.AuditHead != state.Audit.Head() {
		return fmt.Errorf("恢复元数据 audit_head 不一致")
	}
	digest, err := contentDigest(state)
	if err != nil {
		return err
	}
	if digest != metadata.ContentDigest {
		return fmt.Errorf("快照内容摘要不一致")
	}
	return nil
}

func contentDigest(state *State) (string, error) {
	content := struct {
		SchemaVersion int                            `json:"schema_version"`
		Batches       map[string]*domain.SampleBatch `json:"batches"`
		Idempotency   map[string]IdempotencyRecord   `json:"idempotency"`
		ArchiveIndex  map[string]ArchiveRecord       `json:"archive_index"`
		Audit         audit.Chain                    `json:"audit"`
	}{state.SchemaVersion, state.Batches, state.Idempotency, state.ArchiveIndex, state.Audit}
	return audit.Digest(content)
}
