package audit

import (
	"errors"
	"fmt"
	"time"
)

type Chain struct {
	Events []Event `json:"events"`
}

func (c *Chain) Append(batchID, eventType string, revision int64, at time.Time, payload any) (Event, error) {
	payloadHash, err := Digest(payload)
	if err != nil {
		return Event{}, err
	}
	previous := ""
	if len(c.Events) > 0 {
		previous = c.Events[len(c.Events)-1].Hash
	}
	e := Event{Sequence: int64(len(c.Events) + 1), BatchID: batchID, Type: eventType, Revision: revision, OccurredAt: at.UTC(), PayloadHash: payloadHash, PreviousHash: previous}
	e.Hash, err = eventDigest(e)
	if err != nil {
		return Event{}, err
	}
	c.Events = append(c.Events, e)
	return e, nil
}

func (c Chain) Head() string {
	if len(c.Events) == 0 {
		return ""
	}
	return c.Events[len(c.Events)-1].Hash
}

func (c Chain) Contains(hash string) bool {
	if hash == "" {
		return false
	}
	for _, event := range c.Events {
		if event.Hash == hash {
			return true
		}
	}
	return false
}

func (c Chain) PrefixThrough(hash string) ([]Event, error) {
	for index, event := range c.Events {
		if event.Hash == hash {
			result := make([]Event, index+1)
			copy(result, c.Events[:index+1])
			return result, nil
		}
	}
	return nil, fmt.Errorf("审计头 %s 不存在", hash)
}

func (c Chain) Verify() error {
	previous := ""
	for i, e := range c.Events {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("审计事件 %d 字段无效: %w", i+1, err)
		}
		if e.Sequence != int64(i+1) {
			return fmt.Errorf("审计序号在位置 %d 不连续", i)
		}
		if e.PreviousHash != previous {
			return fmt.Errorf("审计前序摘要在序号 %d 不匹配", e.Sequence)
		}
		expected, err := eventDigest(e)
		if err != nil {
			return err
		}
		if expected != e.Hash {
			return fmt.Errorf("审计事件 %d 摘要损坏", e.Sequence)
		}
		previous = e.Hash
	}
	return nil
}

func eventDigest(e Event) (string, error) {
	if e.Type == "" {
		return "", errors.New("审计事件类型不能为空")
	}
	copy := e
	copy.Hash = ""
	return Digest(copy)
}
