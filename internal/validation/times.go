package validation

import (
	"time"
)

func OccurredAt(name string, value, now time.Time) error {
	if value.IsZero() {
		return invalid(name + " 不能为空")
	}
	if value.After(now.Add(5 * time.Minute)) {
		return invalid(name + " 不得晚于当前时间超过 5 分钟")
	}
	return nil
}

func NotBefore(name string, value, earliest time.Time) error {
	if value.Before(earliest) {
		return invalid(name + " 不得早于前序业务事件")
	}
	return nil
}

func Chronological(name string, value, now, earliest time.Time) error {
	if err := OccurredAt(name, value, now); err != nil {
		return err
	}
	return NotBefore(name, value, earliest)
}

func HandoverAt(value, collectedAt, now time.Time) error {
	if value.IsZero() {
		return invalid("handover_at 不能为空")
	}
	if value.Before(collectedAt) {
		return invalid("handover_at 不得早于 collected_at")
	}
	if value.After(now) {
		return invalid("handover_at 不得晚于当前时间")
	}
	return nil
}

func StrictlyAfter(name string, value, previous time.Time) error {
	if !value.After(previous) {
		return invalid(name + " 必须严格晚于上一条记录")
	}
	return nil
}
