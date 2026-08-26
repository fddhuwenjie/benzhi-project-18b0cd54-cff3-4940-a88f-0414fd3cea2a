package storage

import (
	"anaerobic-release/internal/domain"
	"encoding/json"
)

func Replay(state *State, requestID, operation, requestHash string, target any) (bool, error) {
	if requestID == "" {
		return false, domain.NewError("validation_error", "request_id 不能为空", 422)
	}
	record, ok := state.Idempotency[requestID]
	if !ok {
		return false, nil
	}
	if record.Operation != operation || record.RequestHash != requestHash {
		return false, domain.NewError("idempotency_conflict", "request_id 已用于不同请求", 409)
	}
	if err := json.Unmarshal(record.Response, target); err != nil {
		return false, err
	}
	return true, nil
}

func Remember(state *State, requestID, operation, requestHash string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	state.Idempotency[requestID] = IdempotencyRecord{Operation: operation, RequestHash: requestHash, Response: b}
	return nil
}
