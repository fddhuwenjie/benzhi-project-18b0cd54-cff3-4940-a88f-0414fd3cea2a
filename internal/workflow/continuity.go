package workflow

import "anaerobic-release/internal/domain"

func nextCheckpoint(batch *domain.SampleBatch, checkpoint domain.TransferCheckpoint) domain.TransferCheckpoint {
	checkpoint.Sequence = int64(len(batch.Checkpoints) + 1)
	checkpoint.SequenceStart = checkpoint.Sequence == 1
	if len(batch.Checkpoints) > 0 {
		checkpoint.PreviousCheckpointID = batch.Checkpoints[len(batch.Checkpoints)-1].CheckpointID
	}
	return checkpoint
}

func refreshContinuity(batch *domain.SampleBatch) {
	summary := domain.TransferContinuitySummary{TotalCheckpoints: len(batch.Checkpoints)}
	for index := range batch.Checkpoints {
		checkpoint := &batch.Checkpoints[index]
		if !checkpoint.WithinLimits {
			summary.HasLimitBreach = true
		}
		if checkpoint.Sequence != int64(index+1) || checkpoint.SequenceStart != (index == 0) {
			summary.HasBreak = true
		}
		if index > 0 && checkpoint.PreviousCheckpointID != batch.Checkpoints[index-1].CheckpointID {
			summary.HasBreak = true
		}
	}
	if len(batch.Checkpoints) > 0 {
		latest := batch.Checkpoints[len(batch.Checkpoints)-1].RecordedAt
		summary.LatestRecordedAt = &latest
	}
	batch.TransferContinuity = summary
}
