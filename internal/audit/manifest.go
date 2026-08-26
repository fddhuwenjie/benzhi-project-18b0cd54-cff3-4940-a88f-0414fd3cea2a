package audit

import (
	"anaerobic-release/internal/domain"
	"fmt"
	"sort"
)

var manifestCategoryOrder = map[string]int{
	"preservation_plan": 1,
	"checkpoint":        2,
	"deviation":         3,
	"contamination":     4,
	"quality_review":    5,
	"release_signature": 6,
}

func BindEvidence(chain Chain, batchID, category, recordID, evidenceDigest, eventType string, payload any) (domain.EvidenceManifestItem, error) {
	payloadHash, err := Digest(payload)
	if err != nil {
		return domain.EvidenceManifestItem{}, err
	}
	for index := len(chain.Events) - 1; index >= 0; index-- {
		event := chain.Events[index]
		if event.BatchID == batchID && event.Type == eventType && event.PayloadHash == payloadHash {
			return domain.EvidenceManifestItem{Category: category, RecordID: recordID, EvidenceDigest: evidenceDigest, AuditEventHash: event.Hash, AuditEventType: event.Type, AuditRevision: event.Revision}, nil
		}
	}
	return domain.EvidenceManifestItem{}, fmt.Errorf("%s/%s 找不到匹配的审计事件", category, recordID)
}

func SortManifest(items []domain.EvidenceManifestItem) {
	sort.Slice(items, func(i, j int) bool {
		left, right := manifestCategoryOrder[items[i].Category], manifestCategoryOrder[items[j].Category]
		if left != right {
			return left < right
		}
		return items[i].RecordID < items[j].RecordID
	})
}

func VerifyManifest(chain Chain, batchID string, items []domain.EvidenceManifestItem) error {
	if len(items) == 0 {
		return fmt.Errorf("证据清单为空")
	}
	copyItems := append([]domain.EvidenceManifestItem(nil), items...)
	SortManifest(copyItems)
	digests := make(map[string]string, len(items))
	for index, item := range items {
		if _, known := manifestCategoryOrder[item.Category]; !known {
			return fmt.Errorf("清单项 %s 使用未知类别 %s", item.RecordID, item.Category)
		}
		if item.RecordID == "" {
			return fmt.Errorf("类别 %s 的业务记录标识为空", item.Category)
		}
		if err := validateHash("evidence_digest", item.EvidenceDigest, false); err != nil {
			return fmt.Errorf("清单项 %s: %w", item.RecordID, err)
		}
		if prior, duplicate := digests[item.EvidenceDigest]; duplicate {
			return fmt.Errorf("清单项 %s 与 %s 重复使用证据摘要", item.RecordID, prior)
		}
		digests[item.EvidenceDigest] = item.RecordID
		if item != copyItems[index] {
			return fmt.Errorf("证据清单顺序不稳定，位置 %d 为 %s", index+1, item.RecordID)
		}
		event, ok := eventByHash(chain, item.AuditEventHash)
		if !ok {
			return fmt.Errorf("清单项 %s 引用的审计事件不存在", item.RecordID)
		}
		if event.BatchID != batchID {
			return fmt.Errorf("清单项 %s 引用了其他批次的审计事件", item.RecordID)
		}
		if event.Type != item.AuditEventType || event.Revision != item.AuditRevision {
			return fmt.Errorf("清单项 %s 的审计事件类型或 revision 不一致", item.RecordID)
		}
	}
	return nil
}

func eventByHash(chain Chain, hash string) (Event, bool) {
	for _, event := range chain.Events {
		if event.Hash == hash {
			return event, true
		}
	}
	return Event{}, false
}
