package storage

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
	"encoding/json"
	"time"
)

type IdempotencyRecord struct {
	Operation   string          `json:"operation"`
	RequestHash string          `json:"request_hash"`
	Response    json.RawMessage `json:"response"`
}

type State struct {
	SchemaVersion int                            `json:"schema_version"`
	Batches       map[string]*domain.SampleBatch `json:"batches"`
	Idempotency   map[string]IdempotencyRecord   `json:"idempotency"`
	ArchiveIndex  map[string]ArchiveRecord       `json:"archive_index"`
	Audit         audit.Chain                    `json:"audit"`
	Recovery      RecoveryMetadata               `json:"recovery"`
}

func newState() *State {
	return &State{SchemaVersion: 1, Batches: map[string]*domain.SampleBatch{}, Idempotency: map[string]IdempotencyRecord{}, ArchiveIndex: map[string]ArchiveRecord{}}
}

type ArchiveRecord struct {
	BatchID         string    `json:"batch_id"`
	CertificateID   string    `json:"certificate_id"`
	Revision        int64     `json:"revision"`
	AuditHead       string    `json:"audit_head"`
	IntegrityDigest string    `json:"integrity_digest"`
	ManifestItems   int       `json:"manifest_items"`
	ArchivedAt      time.Time `json:"archived_at"`
}

type RecoveryMetadata struct {
	Generation    int64     `json:"generation"`
	SavedAt       time.Time `json:"saved_at"`
	BatchCount    int       `json:"batch_count"`
	ArchivedCount int       `json:"archived_count"`
	AuditHead     string    `json:"audit_head"`
	ContentDigest string    `json:"content_digest"`
}
