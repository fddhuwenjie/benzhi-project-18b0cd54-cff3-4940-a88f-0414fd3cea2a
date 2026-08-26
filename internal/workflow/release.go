package workflow

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
	"fmt"
)

type releaseGateCacheEntry struct {
	err error
}

func (s *Service) cachedReleaseGate(batch *domain.SampleBatch) error {
	if cached, ok := s.releaseGates.Load(batch.BatchID); ok {
		return cached.(releaseGateCacheEntry).err
	}
	err := releaseGate(batch)
	s.releaseGates.Store(batch.BatchID, releaseGateCacheEntry{err: err})
	return err
}

func (s *Service) RecordContamination(cmd domain.RecordContaminationCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	if err := validation.Contamination(cmd.Result); err != nil {
		return nil, err
	}
	for n, v := range map[string]string{"tested_by": cmd.TestedBy, "method": cmd.Method} {
		if err := validation.Required(n, v); err != nil {
			return nil, err
		}
	}
	if err := validation.Identifier("tested_by", cmd.TestedBy); err != nil {
		return nil, err
	}
	if err := validation.Digest(cmd.EvidenceDigest); err != nil {
		return nil, err
	}
	hash, _ := requestHash(cmd)
	var result *domain.SampleBatch
	err := s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "contamination", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if err := validation.Mutable(batch, domain.StatusAwaitingTest); err != nil {
				return err
			}
			if err := validation.NoOpenDeviations(batch); err != nil {
				return err
			}
			if err := validation.Chronological("tested_at", cmd.TestedAt, s.now(), batch.CollectedAt); err != nil {
				return err
			}
			if len(batch.ContaminationHistory) > 0 {
				previous := batch.ContaminationHistory[len(batch.ContaminationHistory)-1]
				if err := validation.StrictlyAfter("tested_at", cmd.TestedAt, previous.TestedAt); err != nil {
					return err
				}
			}
			if err := validation.UniqueContaminationEvidence(batch.ContaminationHistory, cmd.EvidenceDigest); err != nil {
				return err
			}
			attempt := domain.ContaminationTest{TestID: deterministicID("contamination", cmd.BatchID, cmd.RequestID), Attempt: len(batch.ContaminationHistory) + 1, RecordedRevision: batch.Revision + 1, Result: cmd.Result, TestedBy: cmd.TestedBy, TestedAt: cmd.TestedAt.UTC(), Method: cmd.Method, EvidenceDigest: cmd.EvidenceDigest}
			batch.ContaminationHistory = append(batch.ContaminationHistory, attempt)
			batch.Contamination = &attempt
			batch.Review = nil
			batch.CurrentReleaseVersion = domain.CurrentReleaseVersion{ContaminationTestID: attempt.TestID, ContaminationAttempt: attempt.Attempt}
			if cmd.Result == "not_detected" {
				if err := validation.Transition(batch, domain.StatusPendingReview); err != nil {
					return err
				}
			} else if err := validation.Transition(batch, domain.StatusAwaitingTest); err != nil {
				return err
			}
			batch.Revision++
			if err := event(state, batch, "contamination.recorded", s.now(), attempt); err != nil {
				return err
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "contamination", hash, result)
		})
	})
	return result, err
}

func (s *Service) Review(cmd domain.ReviewCommand) (*domain.SampleBatch, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	if err := validation.IndependentReviewer(&domain.SampleBatch{}, cmd.ReviewerID); err != nil && cmd.ReviewerID == "" {
		return nil, err
	}
	if cmd.Decision != "approve" && cmd.Decision != "reject" {
		return nil, domain.NewError("validation_error", "decision 仅允许 approve 或 reject", 422)
	}
	if err := validation.Digest(cmd.EvidenceDigest); err != nil {
		return nil, err
	}
	hash, _ := requestHash(cmd)
	var result *domain.SampleBatch
	err := s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "review", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if err := validation.Mutable(batch, domain.StatusPendingReview); err != nil {
				return err
			}
			if err := validation.IndependentReviewer(batch, cmd.ReviewerID); err != nil {
				return err
			}
			if err := validation.Chronological("reviewed_at", cmd.ReviewedAt, s.now(), batch.Contamination.TestedAt); err != nil {
				return err
			}
			if len(batch.ReviewHistory) > 0 {
				previous := batch.ReviewHistory[len(batch.ReviewHistory)-1]
				if err := validation.StrictlyAfter("reviewed_at", cmd.ReviewedAt, previous.ReviewedAt); err != nil {
					return err
				}
			}
			if err := validation.UniqueReviewEvidence(batch.ReviewHistory, cmd.EvidenceDigest); err != nil {
				return err
			}
			attempt := domain.QualityReview{ReviewID: deterministicID("review", cmd.BatchID, cmd.RequestID), Attempt: len(batch.ReviewHistory) + 1, ContaminationTestID: batch.Contamination.TestID, RecordedRevision: batch.Revision + 1, ReviewerID: cmd.ReviewerID, ReviewedAt: cmd.ReviewedAt.UTC(), Decision: cmd.Decision, EvidenceDigest: cmd.EvidenceDigest}
			batch.ReviewHistory = append(batch.ReviewHistory, attempt)
			batch.Review = &attempt
			batch.CurrentReleaseVersion.QualityReviewID = attempt.ReviewID
			batch.CurrentReleaseVersion.QualityReviewAttempt = attempt.Attempt
			if cmd.Decision == "approve" {
				if err := validation.Transition(batch, domain.StatusReviewed); err != nil {
					return err
				}
			} else {
				if err := validation.Transition(batch, domain.StatusAwaitingTest); err != nil {
					return err
				}
			}
			batch.Revision++
			if err := event(state, batch, "quality.reviewed", s.now(), attempt); err != nil {
				return err
			}
			result = batch
			return storage.Remember(state, cmd.RequestID, "review", hash, result)
		})
	})
	return result, err
}

func (s *Service) Release(cmd domain.ReleaseCommand) (*domain.ReleaseCertificate, error) {
	if err := validateMeta(cmd.WriteMeta); err != nil {
		return nil, err
	}
	if err := validation.Digest(cmd.EvidenceDigest); err != nil {
		return nil, err
	}
	if err := validation.Identifier("issuer_id", cmd.IssuerID); err != nil {
		return nil, err
	}
	hash, _ := requestHash(cmd)
	var result *domain.ReleaseCertificate
	err := s.withBatch(cmd.BatchID, func() error {
		return s.store.Update(func(state *storage.State) error {
			if replay, err := storage.Replay(state, cmd.RequestID, "release", hash, &result); replay || err != nil {
				return err
			}
			batch, ok := state.Batches[cmd.BatchID]
			if !ok {
				return domain.NewError("not_found", "样本批次不存在", 404)
			}
			if err := requireRevision(batch, cmd.ExpectedRevision); err != nil {
				return err
			}
			if batch.Status.Archived() {
				return domain.NewError("invalid_state", "批次已归档，只允许读取", 409)
			}
			if err := s.cachedReleaseGate(batch); err != nil {
				return err
			}
			if err := validation.IndependentIssuer(batch, cmd.IssuerID); err != nil {
				return err
			}
			if err := state.Audit.Verify(); err != nil {
				return err
			}
			now := s.now()
			batch.Revision++
			id := cmd.CertificateID
			if id == "" {
				id = deterministicID("certificate", cmd.BatchID, cmd.RequestID)
			}
			if err := validation.Identifier("certificate_id", id); err != nil {
				return err
			}
			for _, archive := range state.ArchiveIndex {
				if archive.CertificateID == id {
					return domain.NewError("already_exists", "certificate_id 已用于其他归档", 409)
				}
			}
			authorization := domain.ReleaseAuthorizationAudit{BatchID: batch.BatchID, CertificateID: id, IssuerID: cmd.IssuerID, EvidenceDigest: cmd.EvidenceDigest, Revision: batch.Revision}
			if _, err := state.Audit.Append(batch.BatchID, "release.authorized", batch.Revision, now, authorization); err != nil {
				return err
			}
			manifest, err := buildEvidenceManifest(batch, state.Audit, authorization)
			if err != nil {
				return domain.NewError("release_blocked", "证据清单核验失败："+err.Error(), 409)
			}
			preReleaseHead := state.Audit.Head()
			integrityDigest, err := certificateIntegrityDigest(manifest, "released", batch.Revision, preReleaseHead)
			if err != nil {
				return err
			}
			cert := &domain.ReleaseCertificate{CertificateID: id, BatchID: cmd.BatchID, ContaminationResult: batch.Contamination.Result, ReviewerID: batch.Review.ReviewerID, ReviewedAt: batch.Review.ReviewedAt, Decision: "released", EvidenceDigest: integrityDigest, IntegrityDigest: integrityDigest, EvidenceManifest: manifest, PreReleaseAuditHead: preReleaseHead, IssuerEvidenceDigest: cmd.EvidenceDigest, IssuedBy: cmd.IssuerID, IssuedAt: now}
			eventPayload := domain.ReleaseIssuedAudit{BatchID: batch.BatchID, CertificateID: id, Decision: cert.Decision, Revision: batch.Revision, IntegrityDigest: integrityDigest, EvidenceManifest: manifest}
			ev, err := state.Audit.Append(batch.BatchID, "release.issued", batch.Revision, now, eventPayload)
			if err != nil {
				return err
			}
			cert.AuditHead = ev.Hash
			batch.Certificate = cert
			if err := validation.Transition(batch, domain.StatusArchived); err != nil {
				return err
			}
			batch.ArchivedAt = &now
			state.ArchiveIndex[batch.BatchID] = storage.ArchiveRecord{BatchID: batch.BatchID, CertificateID: cert.CertificateID, Revision: batch.Revision, AuditHead: cert.AuditHead, IntegrityDigest: cert.IntegrityDigest, ManifestItems: len(cert.EvidenceManifest), ArchivedAt: now}
			result = cert
			return storage.Remember(state, cmd.RequestID, "release", hash, result)
		})
	})
	return result, err
}

func releaseGate(batch *domain.SampleBatch) error {
	if batch.Status != domain.StatusReviewed || batch.Contamination == nil || batch.Review == nil {
		return domain.NewError("release_blocked", "尚无可用于放行的污染检测与质量复核配对版本", 409)
	}
	if err := validation.NoOpenDeviations(batch); err != nil {
		return domain.NewError("release_blocked", "存在未闭环偏差", 409)
	}
	current := batch.CurrentReleaseVersion
	if batch.Contamination.Result != "not_detected" || batch.Review.Decision != "approve" || batch.Review.ContaminationTestID != batch.Contamination.TestID || current.ContaminationTestID != batch.Contamination.TestID || current.QualityReviewID != batch.Review.ReviewID {
		return domain.NewError("release_blocked", "最新污染检测与质量复核版本不满足放行门禁", 409)
	}
	return nil
}

func certificateIntegrityDigest(manifest []domain.EvidenceManifestItem, decision string, revision int64, preReleaseAuditHead string) (string, error) {
	summary := struct {
		EvidenceManifest    []domain.EvidenceManifestItem `json:"evidence_manifest"`
		Decision            string                        `json:"decision"`
		Revision            int64                         `json:"revision"`
		PreReleaseAuditHead string                        `json:"pre_release_audit_head"`
	}{manifest, decision, revision, preReleaseAuditHead}
	return audit.Digest(summary)
}

func buildEvidenceManifest(batch *domain.SampleBatch, chain audit.Chain, authorization domain.ReleaseAuthorizationAudit) ([]domain.EvidenceManifestItem, error) {
	if batch.PreservationPlan == nil || batch.HandoverConfirmation == nil || batch.PlanFrozenAt == nil || batch.PreservationPlanSummary == "" {
		return nil, fmt.Errorf("preservation_plan 缺少冻结或交接记录")
	}
	items := make([]domain.EvidenceManifestItem, 0, len(batch.Checkpoints)+len(batch.Deviations)+4)
	planPayload := domain.PlanFreezeAudit{Plan: *batch.PreservationPlan, Handover: *batch.HandoverConfirmation, PlanSummary: batch.PreservationPlanSummary, PlanFrozenAt: *batch.PlanFrozenAt}
	item, err := audit.BindEvidence(chain, batch.BatchID, "preservation_plan", batch.PreservationPlanSummary, batch.HandoverConfirmation.EvidenceDigest, "plan.frozen", planPayload)
	if err != nil {
		return nil, err
	}
	items = append(items, item)
	for _, checkpoint := range batch.Checkpoints {
		item, err = audit.BindEvidence(chain, batch.BatchID, "checkpoint", checkpoint.CheckpointID, checkpoint.EvidenceDigest, "checkpoint.recorded", checkpoint)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	for _, deviation := range batch.Deviations {
		if deviation.ResolvedAt == nil || deviation.ResolutionDigest == "" {
			return nil, fmt.Errorf("deviation/%s 尚未闭环", deviation.DeviationID)
		}
		retest, err := checkpointByID(batch, deviation.RetestCheckpointID)
		if err != nil {
			return nil, err
		}
		payload := domain.DeviationResolutionAudit{Deviation: deviation, Retest: retest}
		item, err = audit.BindEvidence(chain, batch.BatchID, "deviation", deviation.DeviationID, deviation.ResolutionDigest, "deviation.resolved", payload)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if batch.Contamination == nil || batch.Review == nil {
		return nil, fmt.Errorf("缺少当前污染检测或质量复核")
	}
	item, err = audit.BindEvidence(chain, batch.BatchID, "contamination", batch.Contamination.TestID, batch.Contamination.EvidenceDigest, "contamination.recorded", *batch.Contamination)
	if err != nil {
		return nil, err
	}
	items = append(items, item)
	item, err = audit.BindEvidence(chain, batch.BatchID, "quality_review", batch.Review.ReviewID, batch.Review.EvidenceDigest, "quality.reviewed", *batch.Review)
	if err != nil {
		return nil, err
	}
	items = append(items, item)
	item, err = audit.BindEvidence(chain, batch.BatchID, "release_signature", authorization.CertificateID, authorization.EvidenceDigest, "release.authorized", authorization)
	if err != nil {
		return nil, err
	}
	items = append(items, item)
	audit.SortManifest(items)
	if err := audit.VerifyManifest(chain, batch.BatchID, items); err != nil {
		return nil, err
	}
	return items, nil
}

func checkpointByID(batch *domain.SampleBatch, id string) (domain.TransferCheckpoint, error) {
	for _, checkpoint := range batch.Checkpoints {
		if checkpoint.CheckpointID == id {
			return checkpoint, nil
		}
	}
	return domain.TransferCheckpoint{}, fmt.Errorf("checkpoint/%s 不存在", id)
}

func Evidence(batch *domain.SampleBatch, proof []audit.Event) (audit.EvidencePackage, error) {
	p := audit.EvidencePackage{BatchID: batch.BatchID, Revision: batch.Revision, AuditProof: proof, EvidenceManifest: batch.Certificate.EvidenceManifest}
	for _, c := range batch.Checkpoints {
		p.CheckpointDigests = append(p.CheckpointDigests, c.EvidenceDigest)
	}
	for _, d := range batch.Deviations {
		if d.ResolutionDigest != "" {
			p.DeviationDigests = append(p.DeviationDigests, d.ResolutionDigest)
		}
	}
	if batch.Contamination != nil {
		p.ContaminationDigest = batch.Contamination.EvidenceDigest
	}
	if batch.Review != nil {
		p.ReviewDigest = batch.Review.EvidenceDigest
	}
	if batch.Certificate != nil {
		p.AuditHead = batch.Certificate.AuditHead
	}
	if err := audit.VerifyEvidence(p); err != nil {
		return p, err
	}
	return p, nil
}

func verifyArchivedEvidence(batch *domain.SampleBatch, proof []audit.Event) error {
	if batch.Certificate == nil {
		return fmt.Errorf("certificate 缺失")
	}
	certificate := batch.Certificate
	chain := audit.Chain{Events: proof}
	authorization := domain.ReleaseAuthorizationAudit{BatchID: batch.BatchID, CertificateID: certificate.CertificateID, IssuerID: certificate.IssuedBy, EvidenceDigest: certificate.IssuerEvidenceDigest, Revision: batch.Revision}
	expectedManifest, err := buildEvidenceManifest(batch, chain, authorization)
	if err != nil {
		return err
	}
	if err := compareManifest(expectedManifest, certificate.EvidenceManifest); err != nil {
		return err
	}
	expectedDigest, err := certificateIntegrityDigest(expectedManifest, certificate.Decision, batch.Revision, certificate.PreReleaseAuditHead)
	if err != nil {
		return err
	}
	if certificate.IntegrityDigest != expectedDigest || certificate.EvidenceDigest != expectedDigest {
		return fmt.Errorf("certificate 完整性摘要不一致")
	}
	issued := domain.ReleaseIssuedAudit{BatchID: batch.BatchID, CertificateID: certificate.CertificateID, Decision: certificate.Decision, Revision: batch.Revision, IntegrityDigest: certificate.IntegrityDigest, EvidenceManifest: certificate.EvidenceManifest}
	item, err := audit.BindEvidence(chain, batch.BatchID, "release_signature", certificate.CertificateID, certificate.IssuerEvidenceDigest, "release.issued", issued)
	if err != nil {
		return fmt.Errorf("放行事件绑定无效: %w", err)
	}
	if item.AuditEventHash != certificate.AuditHead || chain.Head() != certificate.AuditHead {
		return fmt.Errorf("certificate audit_head 与放行事件不一致")
	}
	return nil
}

func compareManifest(expected, actual []domain.EvidenceManifestItem) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("证据清单项目数量不一致：期望 %d，实际 %d", len(expected), len(actual))
	}
	for index := range expected {
		if expected[index] != actual[index] {
			item := actual[index]
			if item.Category == "" {
				item = expected[index]
			}
			return fmt.Errorf("清单项 %s/%s 与业务记录不一致", item.Category, item.RecordID)
		}
	}
	return nil
}
