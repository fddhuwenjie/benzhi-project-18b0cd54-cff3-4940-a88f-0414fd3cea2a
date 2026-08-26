package domain

type Status string

const (
	StatusDraft         Status = "draft"
	StatusReadyTransfer Status = "ready_transfer"
	StatusAwaitingTest  Status = "awaiting_contamination_test"
	StatusQuarantined   Status = "quarantined"
	StatusPendingReview Status = "pending_review"
	StatusReviewed      Status = "reviewed"
	StatusArchived      Status = "released_archived"
)

func (s Status) Archived() bool { return s == StatusArchived }
