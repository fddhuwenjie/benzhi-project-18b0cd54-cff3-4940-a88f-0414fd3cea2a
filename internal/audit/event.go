package audit

import "time"

type Event struct {
	Sequence     int64     `json:"sequence"`
	BatchID      string    `json:"batch_id"`
	Type         string    `json:"event_type"`
	Revision     int64     `json:"revision"`
	OccurredAt   time.Time `json:"occurred_at"`
	PayloadHash  string    `json:"payload_hash"`
	PreviousHash string    `json:"previous_hash"`
	Hash         string    `json:"hash"`
}
