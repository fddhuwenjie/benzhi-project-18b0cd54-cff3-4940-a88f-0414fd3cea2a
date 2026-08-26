package audit

import (
	"encoding/hex"
	"fmt"
	"strings"
)

var knownEventTypes = map[string]struct{}{
	"batch.created":          {},
	"plan.frozen":            {},
	"checkpoint.recorded":    {},
	"deviation.opened":       {},
	"deviation.resolved":     {},
	"contamination.recorded": {},
	"quality.reviewed":       {},
	"release.authorized":     {},
	"release.issued":         {},
}

func (e Event) Validate() error {
	if e.Sequence < 1 {
		return fmt.Errorf("事件 sequence 必须为正数")
	}
	if strings.TrimSpace(e.BatchID) == "" {
		return fmt.Errorf("事件 batch_id 不能为空")
	}
	if _, known := knownEventTypes[e.Type]; !known {
		return fmt.Errorf("未知事件类型 %s", e.Type)
	}
	if e.Revision < 1 {
		return fmt.Errorf("事件 revision 必须为正数")
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("事件 occurred_at 不能为空")
	}
	if err := validateHash("payload_hash", e.PayloadHash, false); err != nil {
		return err
	}
	if err := validateHash("previous_hash", e.PreviousHash, e.Sequence == 1); err != nil {
		return err
	}
	return validateHash("hash", e.Hash, false)
}

func validateHash(name, value string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	if len(value) != 64 {
		return fmt.Errorf("%s 必须为 64 位 SHA-256 十六进制摘要", name)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s 不是合法十六进制摘要", name)
	}
	return nil
}

func (c Chain) ReleaseHead(batchID string, revision int64) (Event, error) {
	for index := len(c.Events) - 1; index >= 0; index-- {
		event := c.Events[index]
		if event.BatchID == batchID && event.Type == "release.issued" && event.Revision == revision {
			return event, nil
		}
	}
	return Event{}, fmt.Errorf("找不到批次 %s revision=%d 的放行事件", batchID, revision)
}
