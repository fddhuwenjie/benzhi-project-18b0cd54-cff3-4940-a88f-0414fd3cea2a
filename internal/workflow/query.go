package workflow

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
)

func (s *Service) Batch(id string) (*domain.SampleBatch, error) { return s.store.Batch(id) }

type EvidenceResult struct {
	Package          audit.EvidencePackage         `json:"package"`
	IntegrityDigest  string                        `json:"integrity_digest"`
	Certificate      *domain.ReleaseCertificate    `json:"certificate"`
	EvidenceManifest []domain.EvidenceManifestItem `json:"evidence_manifest"`
	Verification     string                        `json:"verification"`
}

func (s *Service) Evidence(id string) (*EvidenceResult, error) {
	head, err := s.store.AuditHead()
	if err != nil {
		return nil, err
	}
	cacheKey := evidenceCacheKey(id, head)
	if cached, ok := s.evidenceCache.Load(cacheKey); ok {
		if result, ok := cached.(*EvidenceResult); ok && evidenceBelongsToBatch(result, id) {
			return result, nil
		}
	}
	batch, proof, err := s.store.ArchivedEvidence(id)
	if err != nil {
		return nil, err
	}
	if err := verifyArchivedEvidence(batch, proof); err != nil {
		return nil, domain.NewError("integrity_error", "归档证据核验失败："+err.Error(), 409)
	}
	p, err := Evidence(batch, proof)
	if err != nil {
		return nil, domain.NewError("integrity_error", "归档证据核验失败："+err.Error(), 409)
	}
	result := &EvidenceResult{Package: p, IntegrityDigest: batch.Certificate.IntegrityDigest, Certificate: batch.Certificate, EvidenceManifest: batch.Certificate.EvidenceManifest, Verification: "verified"}
	actual, _ := s.evidenceCache.LoadOrStore(cacheKey, result)
	if stored, ok := actual.(*EvidenceResult); ok && evidenceBelongsToBatch(stored, id) {
		return stored, nil
	}
	return result, nil
}

func evidenceCacheKey(batchID, auditHead string) string {
	return batchID + "\x00" + auditHead
}

func evidenceBelongsToBatch(result *EvidenceResult, batchID string) bool {
	return result != nil && result.Package.BatchID == batchID && result.Certificate != nil && result.Certificate.BatchID == batchID
}
