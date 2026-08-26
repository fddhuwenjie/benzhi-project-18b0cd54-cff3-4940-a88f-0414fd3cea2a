package domain

import "time"

type WriteMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}
type CreateBatchCommand struct {
	WriteMeta
	BatchID, SiteCode, StratumReference, CollectorID string
	CollectedAt                                      time.Time
	BaselineOxygenPPM, BaselineTemperatureC          float64
}
type FreezePlanCommand struct {
	WriteMeta
	BatchID                string
	Plan                   PreservationPlan
	HandoverAt             time.Time
	HandoverEvidenceDigest string
}
type AddCheckpointCommand struct {
	WriteMeta
	BatchID, CheckpointID, RecordedBy string
	RecordedAt                        time.Time
	OxygenPPM, TemperatureC           float64
	SealIntact                        bool
	LocationNote, EvidenceDigest      string
}
type ResolveDeviationCommand struct {
	WriteMeta
	BatchID, DeviationID, CorrectiveAction, ResolvedBy, ResolutionDigest string
	Retest                                                               AddCheckpointCommand
}
type RecordContaminationCommand struct {
	WriteMeta
	BatchID, Result, TestedBy, Method, EvidenceDigest string
	TestedAt                                          time.Time
}
type ReviewCommand struct {
	WriteMeta
	BatchID, ReviewerID, Decision, EvidenceDigest string
	ReviewedAt                                    time.Time
}
type ReleaseCommand struct {
	WriteMeta
	BatchID, IssuerID, CertificateID, EvidenceDigest string
}
