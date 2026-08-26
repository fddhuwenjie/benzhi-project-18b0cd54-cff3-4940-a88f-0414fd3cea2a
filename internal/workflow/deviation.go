package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
)

func (s *Service) ResolveDeviation(cmd domain.ResolveDeviationCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{"batch_id": cmd.BatchID, "deviation_id": cmd.DeviationID, "corrective_action": cmd.CorrectiveAction, "resolved_by": cmd.ResolvedBy} {
		if err := validation.Required(name, value); err != nil {
			return nil, err
		}
	}
	if err := validation.Digest(cmd.ResolutionDigest); err != nil {
		return nil, err
	}
	if err := validation.Digest(cmd.Retest.EvidenceDigest); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{"deviation_id": cmd.DeviationID, "resolved_by": cmd.ResolvedBy, "retest.recorded_by": cmd.Retest.RecordedBy} {
		if err := validation.Identifier(name, value); err != nil {
			return nil, err
		}
	}
	for name, value := range map[string]string{"retest.recorded_by": cmd.Retest.RecordedBy, "retest.location_note": cmd.Retest.LocationNote} {
		if err := validation.Required(name, value); err != nil {
			return nil, err
		}
	}
	hash, err := requestHash(cmd)
	if err != nil {
		return nil, err
	}
	var result *domain.SampleBatch
	err = s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "resolve_deviation", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if err := validation.Mutable(batch, domain.StatusQuarantined); err != nil {
				return err
			}
			deviation, err := findDeviation(batch, cmd.DeviationID)
			if err != nil {
				return err
			}
			if deviation.ResolvedAt != nil {
				return domain.NewError("invalid_state", "偏差已经闭环", 409)
			}
			if err := validation.OccurredAt("retest.recorded_at", cmd.Retest.RecordedAt, s.now()); err != nil {
				return err
			}
			within, err := validation.Checkpoint(*batch.PreservationPlan, cmd.Retest.OxygenPPM, cmd.Retest.TemperatureC, cmd.Retest.SealIntact)
			if err != nil {
				return err
			}
			if !within {
				return domain.NewError("retest_failed", "复测仍超出冻结方案阈值，偏差保持隔离", 422)
			}
			checkpointID := cmd.Retest.CheckpointID
			if checkpointID == "" {
				checkpointID = deterministicID("retest", cmd.BatchID, cmd.RequestID)
			}
			if err := validation.Identifier("retest.checkpoint_id", checkpointID); err != nil {
				return err
			}
			if err := validation.CheckpointContinuity(batch, checkpointID, cmd.Retest.EvidenceDigest, cmd.Retest.RecordedAt); err != nil {
				return err
			}
			retest := domain.TransferCheckpoint{CheckpointID: checkpointID, BatchID: cmd.BatchID, RecordedBy: cmd.Retest.RecordedBy, RecordedAt: cmd.Retest.RecordedAt.UTC(), OxygenPPM: cmd.Retest.OxygenPPM, TemperatureC: cmd.Retest.TemperatureC, SealIntact: cmd.Retest.SealIntact, LocationNote: cmd.Retest.LocationNote, EvidenceDigest: cmd.Retest.EvidenceDigest, WithinLimits: true}
			retest = nextCheckpoint(batch, retest)
			now := s.now()
			batch.Checkpoints = append(batch.Checkpoints, retest)
			refreshContinuity(batch)
			deviation.CorrectiveAction = cmd.CorrectiveAction
			deviation.RetestCheckpointID = checkpointID
			deviation.ResolvedBy = cmd.ResolvedBy
			deviation.ResolvedAt = &now
			deviation.ResolutionDigest = cmd.ResolutionDigest
			if err := validation.Transition(batch, domain.StatusAwaitingTest); err != nil {
				return err
			}
			batch.Revision++
			if err := event(state, batch, "checkpoint.recorded", now, retest); err != nil {
				return err
			}
			payload := domain.DeviationResolutionAudit{Deviation: *deviation, Retest: retest}
			if err := event(state, batch, "deviation.resolved", now, payload); err != nil {
				return err
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "resolve_deviation", hash, result)
		})
	})
	return result, err
}
