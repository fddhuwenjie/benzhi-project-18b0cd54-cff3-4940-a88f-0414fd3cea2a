package validation

import "anaerobic-release/internal/domain"

func Mutable(b *domain.SampleBatch, allowed ...domain.Status) error {
	if b.Status.Archived() {
		return state("批次已归档，只允许读取")
	}
	for _, s := range allowed {
		if b.Status == s {
			return nil
		}
	}
	return state("当前批次状态不允许执行此操作")
}

func NoOpenDeviations(b *domain.SampleBatch) error {
	for _, d := range b.Deviations {
		if d.ResolvedAt == nil {
			return state("存在未闭环偏差，不能进入污染检测或放行")
		}
	}
	return nil
}

var transitions = map[domain.Status]map[domain.Status]struct{}{
	domain.StatusDraft:         {domain.StatusReadyTransfer: {}},
	domain.StatusReadyTransfer: {domain.StatusAwaitingTest: {}, domain.StatusQuarantined: {}},
	domain.StatusAwaitingTest:  {domain.StatusAwaitingTest: {}, domain.StatusQuarantined: {}, domain.StatusPendingReview: {}},
	domain.StatusQuarantined:   {domain.StatusAwaitingTest: {}},
	domain.StatusPendingReview: {domain.StatusReviewed: {}, domain.StatusAwaitingTest: {}},
	domain.StatusReviewed:      {domain.StatusArchived: {}},
}

func Transition(batch *domain.SampleBatch, next domain.Status) error {
	if batch.Status.Archived() {
		return state("已归档批次不能再转换状态")
	}
	allowed, exists := transitions[batch.Status]
	if !exists {
		return state("当前状态没有后续转换")
	}
	if _, exists := allowed[next]; !exists {
		return state("不允许从 " + string(batch.Status) + " 转换到 " + string(next))
	}
	batch.Status = next
	return nil
}
