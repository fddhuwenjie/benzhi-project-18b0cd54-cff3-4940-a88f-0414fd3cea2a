package storage

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
	"encoding/json"
)

func (s *Store) Batch(id string) (*domain.SampleBatch, error) {
	var result *domain.SampleBatch
	err := s.View(func(state *State) error {
		batch, ok := state.Batches[id]
		if !ok {
			return domain.NewError("not_found", "样本批次不存在", 404)
		}
		b, _ := json.Marshal(batch)
		return json.Unmarshal(b, &result)
	})
	return result, err
}

func (s *Store) ArchivedEvidence(id string) (*domain.SampleBatch, []audit.Event, error) {
	var batch *domain.SampleBatch
	var proof []audit.Event
	err := s.View(func(state *State) error {
		source, ok := state.Batches[id]
		if !ok {
			return domain.NewError("not_found", "样本批次不存在", 404)
		}
		record, ok := state.ArchiveIndex[id]
		if !ok || !source.Status.Archived() {
			return domain.NewError("invalid_state", "证据包仅在归档后可读取", 409)
		}
		if err := verifyArchive(source, record, state.Audit); err != nil {
			return domain.NewError("integrity_error", "归档索引核验失败："+err.Error(), 409)
		}
		encoded, err := json.Marshal(source)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(encoded, &batch); err != nil {
			return err
		}
		proof, err = s.archivedProof(id, record.AuditHead, state.Audit)
		return err
	})
	return batch, proof, err
}

func (s *Store) archivedProof(batchID, auditHead string, chain audit.Chain) ([]audit.Event, error) {
	s.proofMu.Lock()
	defer s.proofMu.Unlock()
	if cached, ok := s.archiveProofCache[batchID]; ok {
		proof := make([]audit.Event, len(cached))
		copy(proof, cached)
		return proof, nil
	}
	proof, err := chain.PrefixThrough(auditHead)
	if err != nil {
		return nil, err
	}
	cached := make([]audit.Event, len(proof))
	copy(cached, proof)
	s.archiveProofCache[batchID] = cached
	return proof, nil
}

func (s *Store) AuditHead() (string, error) {
	var head string
	err := s.View(func(state *State) error { head = state.Audit.Head(); return state.Audit.Verify() })
	return head, err
}
