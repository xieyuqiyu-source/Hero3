package modifier

import "time"

const (
	ModeFlat            = "flat"
	ModePercentAdd      = "percentAdd"
	ModePercentMultiply = "percentMultiply"
)

// Modifier 表示一个属性修改器（所有加成的统一表达）。
type Modifier struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
	Mode  string  `json:"mode"`
}

// Source 所有能提供加成的来源都实现此接口。
type Source interface {
	Modifiers(now time.Time) []Modifier
	SourceName() string
	ExpiresAt() []time.Time
}

func IsValidMode(mode string) bool {
	switch mode {
	case ModeFlat, ModePercentAdd, ModePercentMultiply:
		return true
	default:
		return false
	}
}

// ComputeAttribute 根据所有来源计算最终属性值。
// 公式：(base + flatSum) * (1 + percentAddSum) * percentMultiply1 * percentMultiply2 ...
func ComputeAttribute(base float64, key string, sources ...Source) float64 {
	return ComputeAttributeAt(base, key, time.Now(), sources...)
}

// ComputeAttributeAt 指定时间点计算最终属性值。
func ComputeAttributeAt(base float64, key string, now time.Time, sources ...Source) float64 {
	var flatSum float64
	var percentAddSum float64
	multipliers := make([]float64, 0, 4)

	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, mod := range src.Modifiers(now) {
			if mod.Key != key {
				continue
			}
			switch mod.Mode {
			case ModeFlat:
				flatSum += mod.Value
			case ModePercentAdd:
				percentAddSum += mod.Value
			case ModePercentMultiply:
				multipliers = append(multipliers, 1+mod.Value)
			}
		}
	}

	result := (base + flatSum) * (1 + percentAddSum)
	for _, multiplier := range multipliers {
		result *= multiplier
	}
	return result
}

// ComputeIntAttribute 整数版本（向下取整）。
func ComputeIntAttribute(base int, key string, sources ...Source) int {
	return int(ComputeAttribute(float64(base), key, sources...))
}

// ComputeIntAttributeAt 整数版本，指定时间点。
func ComputeIntAttributeAt(base int, key string, now time.Time, sources ...Source) int {
	return int(ComputeAttributeAt(float64(base), key, now, sources...))
}
