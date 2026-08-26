package domain

import "time"

type PreservationPlan struct {
	ContainerID     string  `json:"container_id"`
	SealMethod      string  `json:"seal_method"`
	CultureTarget   string  `json:"culture_target"`
	CustodianID     string  `json:"custodian_id"`
	MaxOxygenPPM    float64 `json:"max_oxygen_ppm"`
	MinTemperatureC float64 `json:"min_temperature_c"`
	MaxTemperatureC float64 `json:"max_temperature_c"`
}

type HandoverConfirmation struct {
	HandoverAt     time.Time `json:"handover_at"`
	EvidenceDigest string    `json:"evidence_digest"`
	CustodianID    string    `json:"custodian_id"`
}

type PlanFreezeAudit struct {
	Plan         PreservationPlan     `json:"plan"`
	Handover     HandoverConfirmation `json:"handover"`
	PlanSummary  string               `json:"plan_summary"`
	PlanFrozenAt time.Time            `json:"plan_frozen_at"`
}

type SampleBatch struct {
	BatchID                 string                    `json:"batch_id"`
	SiteCode                string                    `json:"site_code"`
	StratumReference        string                    `json:"stratum_reference"`
	CollectorID             string                    `json:"collector_id"`
	CollectedAt             time.Time                 `json:"collected_at"`
	BaselineOxygenPPM       float64                   `json:"baseline_oxygen_ppm"`
	BaselineTemperatureC    float64                   `json:"baseline_temperature_c"`
	PreservationPlan        *PreservationPlan         `json:"preservation_plan,omitempty"`
	HandoverConfirmation    *HandoverConfirmation     `json:"handover_confirmation,omitempty"`
	PreservationPlanSummary string                    `json:"preservation_plan_summary,omitempty"`
	PlanFrozenAt            *time.Time                `json:"plan_frozen_at,omitempty"`
	Status                  Status                    `json:"status"`
	Revision                int64                     `json:"revision"`
	CreatedAt               time.Time                 `json:"created_at"`
	ArchivedAt              *time.Time                `json:"archived_at,omitempty"`
	Checkpoints             []TransferCheckpoint      `json:"checkpoints"`
	TransferContinuity      TransferContinuitySummary `json:"transfer_continuity"`
	Deviations              []DeviationCase           `json:"deviations"`
	Contamination           *ContaminationTest        `json:"contamination,omitempty"`
	ContaminationHistory    []ContaminationTest       `json:"contamination_history"`
	Review                  *QualityReview            `json:"review,omitempty"`
	ReviewHistory           []QualityReview           `json:"review_history"`
	CurrentReleaseVersion   CurrentReleaseVersion     `json:"current_release_version"`
	Certificate             *ReleaseCertificate       `json:"certificate,omitempty"`
}

type TransferCheckpoint struct {
	Sequence             int64     `json:"sequence"`
	PreviousCheckpointID string    `json:"previous_checkpoint_id,omitempty"`
	SequenceStart        bool      `json:"sequence_start"`
	CheckpointID         string    `json:"checkpoint_id"`
	BatchID              string    `json:"batch_id"`
	RecordedBy           string    `json:"recorded_by"`
	RecordedAt           time.Time `json:"recorded_at"`
	OxygenPPM            float64   `json:"oxygen_ppm"`
	TemperatureC         float64   `json:"temperature_c"`
	SealIntact           bool      `json:"seal_intact"`
	LocationNote         string    `json:"location_note"`
	EvidenceDigest       string    `json:"evidence_digest"`
	WithinLimits         bool      `json:"within_limits"`
}

type TransferContinuitySummary struct {
	LatestRecordedAt *time.Time `json:"latest_recorded_at,omitempty"`
	TotalCheckpoints int        `json:"total_checkpoints"`
	HasBreak         bool       `json:"has_break"`
	HasLimitBreach   bool       `json:"has_limit_breach"`
}

type DeviationCase struct {
	DeviationID        string     `json:"deviation_id"`
	BatchID            string     `json:"batch_id"`
	CheckpointID       string     `json:"checkpoint_id"`
	DetectedAt         time.Time  `json:"detected_at"`
	Severity           string     `json:"severity"`
	ContainmentAction  string     `json:"containment_action"`
	CorrectiveAction   string     `json:"corrective_action,omitempty"`
	RetestCheckpointID string     `json:"retest_checkpoint_id,omitempty"`
	ResolvedBy         string     `json:"resolved_by,omitempty"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	ResolutionDigest   string     `json:"resolution_digest,omitempty"`
}

type ContaminationTest struct {
	TestID           string    `json:"test_id"`
	Attempt          int       `json:"attempt"`
	RecordedRevision int64     `json:"recorded_revision"`
	Result           string    `json:"result"`
	TestedBy         string    `json:"tested_by"`
	TestedAt         time.Time `json:"tested_at"`
	Method           string    `json:"method"`
	EvidenceDigest   string    `json:"evidence_digest"`
}

type QualityReview struct {
	ReviewID            string    `json:"review_id"`
	Attempt             int       `json:"attempt"`
	ContaminationTestID string    `json:"contamination_test_id"`
	RecordedRevision    int64     `json:"recorded_revision"`
	ReviewerID          string    `json:"reviewer_id"`
	ReviewedAt          time.Time `json:"reviewed_at"`
	Decision            string    `json:"decision"`
	EvidenceDigest      string    `json:"evidence_digest"`
}

type CurrentReleaseVersion struct {
	ContaminationTestID  string `json:"contamination_test_id,omitempty"`
	ContaminationAttempt int    `json:"contamination_attempt,omitempty"`
	QualityReviewID      string `json:"quality_review_id,omitempty"`
	QualityReviewAttempt int    `json:"quality_review_attempt,omitempty"`
}

type EvidenceManifestItem struct {
	Category       string `json:"category"`
	RecordID       string `json:"record_id"`
	EvidenceDigest string `json:"evidence_digest"`
	AuditEventHash string `json:"audit_event_hash"`
	AuditEventType string `json:"audit_event_type"`
	AuditRevision  int64  `json:"audit_revision"`
}

type ReleaseCertificate struct {
	CertificateID        string                 `json:"certificate_id"`
	BatchID              string                 `json:"batch_id"`
	ContaminationResult  string                 `json:"contamination_result"`
	ReviewerID           string                 `json:"reviewer_id"`
	ReviewedAt           time.Time              `json:"reviewed_at"`
	Decision             string                 `json:"decision"`
	EvidenceDigest       string                 `json:"evidence_digest"`
	IntegrityDigest      string                 `json:"integrity_digest"`
	EvidenceManifest     []EvidenceManifestItem `json:"evidence_manifest"`
	PreReleaseAuditHead  string                 `json:"pre_release_audit_head"`
	IssuerEvidenceDigest string                 `json:"issuer_evidence_digest"`
	AuditHead            string                 `json:"audit_head"`
	IssuedBy             string                 `json:"issued_by"`
	IssuedAt             time.Time              `json:"issued_at"`
}

type DeviationResolutionAudit struct {
	Deviation DeviationCase      `json:"deviation"`
	Retest    TransferCheckpoint `json:"retest"`
}

type ReleaseAuthorizationAudit struct {
	BatchID        string `json:"batch_id"`
	CertificateID  string `json:"certificate_id"`
	IssuerID       string `json:"issuer_id"`
	EvidenceDigest string `json:"evidence_digest"`
	Revision       int64  `json:"revision"`
}

type ReleaseIssuedAudit struct {
	BatchID          string                 `json:"batch_id"`
	CertificateID    string                 `json:"certificate_id"`
	Decision         string                 `json:"decision"`
	Revision         int64                  `json:"revision"`
	IntegrityDigest  string                 `json:"integrity_digest"`
	EvidenceManifest []EvidenceManifestItem `json:"evidence_manifest"`
}
