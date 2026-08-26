package validation

import "anaerobic-release/internal/domain"

func IndependentReviewer(b *domain.SampleBatch, reviewer string) error {
	if err := Identifier("reviewer_id", reviewer); err != nil {
		return err
	}
	if reviewer == b.CollectorID {
		return invalid("复核员不得与采样员相同")
	}
	if b.Contamination != nil && reviewer == b.Contamination.TestedBy {
		return invalid("复核员不得与污染检测人员相同")
	}
	for _, d := range b.Deviations {
		if reviewer == d.ResolvedBy {
			return invalid("复核员不得与偏差纠正人员相同")
		}
	}
	return nil
}

func IndependentCustodian(b *domain.SampleBatch, custodian string) error {
	if err := Identifier("custodian_id", custodian); err != nil {
		return err
	}
	if custodian == b.CollectorID {
		return invalid("custodian_id 不得与 collector_id 相同")
	}
	return nil
}

func IndependentIssuer(b *domain.SampleBatch, issuer string) error {
	if err := Identifier("issuer_id", issuer); err != nil {
		return err
	}
	if issuer == b.CollectorID || (b.Review != nil && issuer == b.Review.ReviewerID) {
		return invalid("签发员须独立于采样员和复核员")
	}
	return nil
}
