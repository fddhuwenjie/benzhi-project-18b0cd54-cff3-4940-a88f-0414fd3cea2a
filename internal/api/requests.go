package api

import (
	"anaerobic-release/internal/domain"
	"time"
)

type createBatchRequest struct {
	BatchID              string    `json:"batch_id,omitempty"`
	SiteCode             string    `json:"site_code"`
	StratumReference     string    `json:"stratum_reference"`
	CollectorID          string    `json:"collector_id"`
	CollectedAt          time.Time `json:"collected_at"`
	BaselineOxygenPPM    float64   `json:"baseline_oxygen_ppm"`
	BaselineTemperatureC float64   `json:"baseline_temperature_c"`
	ExpectedRevision     int64     `json:"expected_revision"`
}

func (v createBatchRequest) command(id string) domain.CreateBatchCommand {
	return domain.CreateBatchCommand{WriteMeta: domain.WriteMeta{RequestID: id, ExpectedRevision: v.ExpectedRevision}, BatchID: v.BatchID, SiteCode: v.SiteCode, StratumReference: v.StratumReference, CollectorID: v.CollectorID, CollectedAt: v.CollectedAt, BaselineOxygenPPM: v.BaselineOxygenPPM, BaselineTemperatureC: v.BaselineTemperatureC}
}

type freezePlanRequest struct {
	ExpectedRevision       int64     `json:"expected_revision"`
	HandoverAt             time.Time `json:"handover_at"`
	HandoverEvidenceDigest string    `json:"handover_evidence_digest"`
	ContainerID            string    `json:"container_id"`
	SealMethod             string    `json:"seal_method"`
	CultureTarget          string    `json:"culture_target"`
	CustodianID            string    `json:"custodian_id"`
	MaxOxygenPPM           float64   `json:"max_oxygen_ppm"`
	MinTemperatureC        float64   `json:"min_temperature_c"`
	MaxTemperatureC        float64   `json:"max_temperature_c"`
}

func (v freezePlanRequest) command(batchID, id string) domain.FreezePlanCommand {
	return domain.FreezePlanCommand{WriteMeta: domain.WriteMeta{RequestID: id, ExpectedRevision: v.ExpectedRevision}, BatchID: batchID, HandoverAt: v.HandoverAt, HandoverEvidenceDigest: v.HandoverEvidenceDigest, Plan: domain.PreservationPlan{ContainerID: v.ContainerID, SealMethod: v.SealMethod, CultureTarget: v.CultureTarget, CustodianID: v.CustodianID, MaxOxygenPPM: v.MaxOxygenPPM, MinTemperatureC: v.MinTemperatureC, MaxTemperatureC: v.MaxTemperatureC}}
}

type checkpointRequest struct {
	ExpectedRevision int64     `json:"expected_revision"`
	CheckpointID     string    `json:"checkpoint_id,omitempty"`
	RecordedBy       string    `json:"recorded_by"`
	RecordedAt       time.Time `json:"recorded_at"`
	OxygenPPM        float64   `json:"oxygen_ppm"`
	TemperatureC     float64   `json:"temperature_c"`
	SealIntact       bool      `json:"seal_intact"`
	LocationNote     string    `json:"location_note"`
	EvidenceDigest   string    `json:"evidence_digest"`
}

func (v checkpointRequest) command(batchID, id string) domain.AddCheckpointCommand {
	return domain.AddCheckpointCommand{WriteMeta: domain.WriteMeta{RequestID: id, ExpectedRevision: v.ExpectedRevision}, BatchID: batchID, CheckpointID: v.CheckpointID, RecordedBy: v.RecordedBy, RecordedAt: v.RecordedAt, OxygenPPM: v.OxygenPPM, TemperatureC: v.TemperatureC, SealIntact: v.SealIntact, LocationNote: v.LocationNote, EvidenceDigest: v.EvidenceDigest}
}

type resolveRequest struct {
	ExpectedRevision int64             `json:"expected_revision"`
	CorrectiveAction string            `json:"corrective_action"`
	ResolvedBy       string            `json:"resolved_by"`
	ResolutionDigest string            `json:"resolution_digest"`
	Retest           checkpointRequest `json:"retest"`
}
type contaminationRequest struct {
	ExpectedRevision int64     `json:"expected_revision"`
	Result           string    `json:"result"`
	TestedBy         string    `json:"tested_by"`
	TestedAt         time.Time `json:"tested_at"`
	Method           string    `json:"method"`
	EvidenceDigest   string    `json:"evidence_digest"`
}
type reviewRequest struct {
	ExpectedRevision int64     `json:"expected_revision"`
	ReviewerID       string    `json:"reviewer_id"`
	ReviewedAt       time.Time `json:"reviewed_at"`
	Decision         string    `json:"decision"`
	EvidenceDigest   string    `json:"evidence_digest"`
}
type releaseRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	IssuerID         string `json:"issuer_id"`
	CertificateID    string `json:"certificate_id,omitempty"`
	EvidenceDigest   string `json:"evidence_digest"`
}
