package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
)

func (s *Service) CreateBatch(cmd domain.CreateBatchCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{"site_code": cmd.SiteCode, "stratum_reference": cmd.StratumReference, "collector_id": cmd.CollectorID} {
		if err := validation.Required(name, value); err != nil {
			return nil, err
		}
	}
	if err := validation.Baseline(cmd.BaselineOxygenPPM, cmd.BaselineTemperatureC); err != nil {
		return nil, err
	}
	if err := validation.OccurredAt("collected_at", cmd.CollectedAt, s.now()); err != nil {
		return nil, err
	}
	if cmd.BatchID == "" {
		cmd.BatchID = deterministicID("batch", cmd.SiteCode, cmd.StratumReference, cmd.RequestID)
	}
	if err := validation.Identifier("batch_id", cmd.BatchID); err != nil {
		return nil, err
	}
	if err := validation.Identifier("collector_id", cmd.CollectorID); err != nil {
		return nil, err
	}
	if cmd.ExpectedRevision != 0 {
		return nil, domain.RevisionError(cmd.ExpectedRevision, 0)
	}
	hash, err := requestHash(cmd)
	if err != nil {
		return nil, err
	}
	var result *domain.SampleBatch
	err = s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "create_batch", hash, &result); replay || err != nil {
				return err
			}
			if _, exists := state.Batches[cmd.BatchID]; exists {
				return domain.NewError("already_exists", "batch_id 已存在", 409)
			}
			now := s.now()
			batch := &domain.SampleBatch{BatchID: cmd.BatchID, SiteCode: cmd.SiteCode, StratumReference: cmd.StratumReference, CollectorID: cmd.CollectorID, CollectedAt: cmd.CollectedAt.UTC(), BaselineOxygenPPM: cmd.BaselineOxygenPPM, BaselineTemperatureC: cmd.BaselineTemperatureC, Status: domain.StatusDraft, Revision: 1, CreatedAt: now, Checkpoints: []domain.TransferCheckpoint{}, Deviations: []domain.DeviationCase{}, ContaminationHistory: []domain.ContaminationTest{}, ReviewHistory: []domain.QualityReview{}}
			state.Batches[batch.BatchID] = batch
			if err := event(state, batch, "batch.created", now, cmd); err != nil {
				return err
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "create_batch", hash, result)
		})
	})
	return result, err
}
