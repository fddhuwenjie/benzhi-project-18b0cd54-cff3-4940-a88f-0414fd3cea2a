package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
)

func (s *Service) AddCheckpoint(cmd domain.AddCheckpointCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{"batch_id": cmd.BatchID, "recorded_by": cmd.RecordedBy, "location_note": cmd.LocationNote} {
		if err := validation.Required(name, value); err != nil {
			return nil, err
		}
	}
	if err := validation.Digest(cmd.EvidenceDigest); err != nil {
		return nil, err
	}
	if cmd.CheckpointID == "" {
		cmd.CheckpointID = deterministicID("checkpoint", cmd.BatchID, cmd.RequestID)
	}
	if err := validation.Identifier("checkpoint_id", cmd.CheckpointID); err != nil {
		return nil, err
	}
	if err := validation.Identifier("recorded_by", cmd.RecordedBy); err != nil {
		return nil, err
	}
	hash, err := requestHash(cmd)
	if err != nil {
		return nil, err
	}
	var result *domain.SampleBatch
	err = s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "add_checkpoint", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if err := validation.Mutable(batch, domain.StatusReadyTransfer, domain.StatusAwaitingTest); err != nil {
				return err
			}
			if err := validation.OccurredAt("recorded_at", cmd.RecordedAt, s.now()); err != nil {
				return err
			}
			if err := validation.CheckpointContinuity(batch, cmd.CheckpointID, cmd.EvidenceDigest, cmd.RecordedAt); err != nil {
				return err
			}
			within, err := validation.Checkpoint(*batch.PreservationPlan, cmd.OxygenPPM, cmd.TemperatureC, cmd.SealIntact)
			if err != nil {
				return err
			}
			cp := domain.TransferCheckpoint{CheckpointID: cmd.CheckpointID, BatchID: cmd.BatchID, RecordedBy: cmd.RecordedBy, RecordedAt: cmd.RecordedAt.UTC(), OxygenPPM: cmd.OxygenPPM, TemperatureC: cmd.TemperatureC, SealIntact: cmd.SealIntact, LocationNote: cmd.LocationNote, EvidenceDigest: cmd.EvidenceDigest, WithinLimits: within}
			cp = nextCheckpoint(batch, cp)
			batch.Checkpoints = append(batch.Checkpoints, cp)
			refreshContinuity(batch)
			batch.Revision++
			if within {
				if err := validation.Transition(batch, domain.StatusAwaitingTest); err != nil {
					return err
				}
			} else {
				if err := validation.Transition(batch, domain.StatusQuarantined); err != nil {
					return err
				}
				batch.Deviations = append(batch.Deviations, domain.DeviationCase{DeviationID: deterministicID("deviation", cmd.BatchID, cmd.CheckpointID), BatchID: cmd.BatchID, CheckpointID: cmd.CheckpointID, DetectedAt: s.now(), Severity: severity(cp), ContainmentAction: "立即隔离并停止培养放行"})
			}
			if err := event(state, batch, "checkpoint.recorded", s.now(), cp); err != nil {
				return err
			}
			if !within {
				if err := event(state, batch, "deviation.opened", s.now(), batch.Deviations[len(batch.Deviations)-1]); err != nil {
					return err
				}
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "add_checkpoint", hash, result)
		})
	})
	return result, err
}

func severity(cp domain.TransferCheckpoint) string {
	if !cp.SealIntact || cp.OxygenPPM > 5000 {
		return "critical"
	}
	return "major"
}
