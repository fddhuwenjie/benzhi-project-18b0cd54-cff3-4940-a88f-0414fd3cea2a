package workflow

import (
	"anaerobic-release/internal/audit"
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/validation"
	"fmt"
	"time"
)

func validateMeta(meta domain.WriteMeta) error {
	if err := validation.Required("request_id", meta.RequestID); err != nil {
		return err
	}
	if meta.ExpectedRevision < 0 {
		return domain.NewError("validation_error", "expected_revision 不得为负数", 422)
	}
	return nil
}

func requireRevision(batch *domain.SampleBatch, expected int64) error {
	if batch.Revision != expected {
		return domain.RevisionError(expected, batch.Revision)
	}
	return nil
}

func requestHash(command any) (string, error) { return audit.Digest(command) }

func deterministicID(prefix string, values ...string) string {
	digest := audit.MustDigest(values)
	return fmt.Sprintf("%s_%s", prefix, digest[:20])
}

func event(state *storage.State, batch *domain.SampleBatch, eventType string, at time.Time, payload any) error {
	_, err := state.Audit.Append(batch.BatchID, eventType, batch.Revision, at, payload)
	return err
}

func findDeviation(batch *domain.SampleBatch, id string) (*domain.DeviationCase, error) {
	for i := range batch.Deviations {
		if batch.Deviations[i].DeviationID == id {
			return &batch.Deviations[i], nil
		}
	}
	return nil, domain.NewError("not_found", "偏差记录不存在", 404)
}
