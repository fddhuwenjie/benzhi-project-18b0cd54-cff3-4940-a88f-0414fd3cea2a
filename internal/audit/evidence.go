package audit

import (
	"anaerobic-release/internal/domain"
	"fmt"
)

type EvidencePackage struct {
	BatchID             string                        `json:"batch_id"`
	Revision            int64                         `json:"revision"`
	CheckpointDigests   []string                      `json:"checkpoint_digests"`
	DeviationDigests    []string                      `json:"deviation_digests"`
	ContaminationDigest string                        `json:"contamination_digest"`
	ReviewDigest        string                        `json:"review_digest"`
	AuditHead           string                        `json:"audit_head"`
	AuditProof          []Event                       `json:"audit_proof"`
	EvidenceManifest    []domain.EvidenceManifestItem `json:"evidence_manifest"`
}

func EvidenceDigest(p EvidencePackage) (string, error) { return Digest(p) }

func VerifyEvidence(p EvidencePackage) error {
	if p.BatchID == "" || p.Revision < 1 {
		return fmt.Errorf("证据包批次标识或 revision 无效")
	}
	if len(p.AuditProof) == 0 {
		return fmt.Errorf("证据包缺少审计证明")
	}
	proof := Chain{Events: p.AuditProof}
	if err := proof.Verify(); err != nil {
		return fmt.Errorf("证据包审计证明无效: %w", err)
	}
	if proof.Head() != p.AuditHead {
		return fmt.Errorf("证据包 audit_head 与证明末端不一致")
	}
	if err := VerifyManifest(proof, p.BatchID, p.EvidenceManifest); err != nil {
		return fmt.Errorf("证据清单无效: %w", err)
	}
	last := p.AuditProof[len(p.AuditProof)-1]
	if last.BatchID != p.BatchID || last.Type != "release.issued" || last.Revision != p.Revision {
		return fmt.Errorf("证据包末端不是对应批次的放行事件")
	}
	return nil
}
