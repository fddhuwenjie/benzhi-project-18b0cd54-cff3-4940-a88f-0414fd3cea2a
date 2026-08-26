package validation

import "anaerobic-release/internal/domain"

const BaselineMaxOxygenPPM = 1000.0

func Baseline(oxygen, temperature float64) error {
	if oxygen < 0 || oxygen > BaselineMaxOxygenPPM {
		return invalid("初始氧浓度必须在 0 到 1000 ppm 之间")
	}
	if temperature < -20 || temperature > 45 {
		return invalid("初始温度必须在 -20 到 45 摄氏度之间")
	}
	return nil
}

func Plan(p domain.PreservationPlan) error {
	for name, value := range map[string]string{"container_id": p.ContainerID, "seal_method": p.SealMethod, "culture_target": p.CultureTarget, "custodian_id": p.CustodianID} {
		if err := Required(name, value); err != nil {
			return err
		}
	}
	if p.MaxOxygenPPM <= 0 || p.MaxOxygenPPM > BaselineMaxOxygenPPM {
		return invalid("max_oxygen_ppm 必须大于 0 且不超过 1000")
	}
	if p.MinTemperatureC < -20 || p.MaxTemperatureC > 45 || p.MinTemperatureC >= p.MaxTemperatureC {
		return invalid("温度上下限无效")
	}
	return nil
}

func PlanBaseline(batch *domain.SampleBatch, plan domain.PreservationPlan) error {
	if batch.BaselineOxygenPPM > plan.MaxOxygenPPM {
		return invalid("baseline_oxygen_ppm 高于 max_oxygen_ppm")
	}
	if batch.BaselineTemperatureC < plan.MinTemperatureC || batch.BaselineTemperatureC > plan.MaxTemperatureC {
		return invalid("baseline_temperature_c 不在保存方案温区内")
	}
	return nil
}

func Checkpoint(p domain.PreservationPlan, oxygen, temperature float64, seal bool) (bool, error) {
	if oxygen < 0 || oxygen > 100000 {
		return false, invalid("oxygen_ppm 超出可测范围")
	}
	if temperature < -50 || temperature > 80 {
		return false, invalid("temperature_c 超出可测范围")
	}
	return seal && oxygen <= p.MaxOxygenPPM && temperature >= p.MinTemperatureC && temperature <= p.MaxTemperatureC, nil
}

func Contamination(result string) error {
	if result != "not_detected" && result != "detected" {
		return invalid("污染检测 result 仅允许 not_detected 或 detected")
	}
	return nil
}
