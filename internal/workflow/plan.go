package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
)

func (s *Service) FreezePlan(cmd domain.FreezePlanCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	if err := validation.Required("batch_id", cmd.BatchID); err != nil {
		return nil, err
	}
	if err := validation.Plan(cmd.Plan); err != nil {
		return nil, err
	}
	if err := validation.Digest(cmd.HandoverEvidenceDigest); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{"container_id": cmd.Plan.ContainerID, "custodian_id": cmd.Plan.CustodianID} {
		if err := validation.Identifier(name, value); err != nil {
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
			if replay, err := storage.Replay(state, cmd.RequestID, "freeze_plan", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if err := validation.Mutable(batch, domain.StatusDraft); err != nil {
				return err
			}
			if err := validation.PlanBaseline(batch, cmd.Plan); err != nil {
				return err
			}
			if err := validation.HandoverAt(cmd.HandoverAt, batch.CollectedAt, s.now()); err != nil {
				return err
			}
			if err := validation.IndependentCustodian(batch, cmd.Plan.CustodianID); err != nil {
				return err
			}
			summary, err := requestHash(cmd.Plan)
			if err != nil {
				return err
			}
			now := s.now()
			handover := &domain.HandoverConfirmation{HandoverAt: cmd.HandoverAt.UTC(), EvidenceDigest: cmd.HandoverEvidenceDigest, CustodianID: cmd.Plan.CustodianID}
			batch.PreservationPlan = &cmd.Plan
			batch.HandoverConfirmation = handover
			batch.PreservationPlanSummary = summary
			batch.PlanFrozenAt = &now
			if err := validation.Transition(batch, domain.StatusReadyTransfer); err != nil {
				return err
			}
			batch.Revision++
			payload := domain.PlanFreezeAudit{Plan: cmd.Plan, Handover: *handover, PlanSummary: summary, PlanFrozenAt: now}
			if err := event(state, batch, "plan.frozen", now, payload); err != nil {
				return err
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "freeze_plan", hash, result)
		})
	})
	return result, err
}
