package validation

import "anaerobic-release/internal/domain"

func invalid(message string) error { return domain.NewError("validation_error", message, 422) }
func state(message string) error   { return domain.NewError("invalid_state", message, 409) }
