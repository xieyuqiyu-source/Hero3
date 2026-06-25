package game

import "time"

// 本文件归口 Buff/DeBuff 状态和 Modifier 管线来源。

// Buff 表示一条加成记录，所有动态加成统一用此结构存储。
type Buff struct {
	ID        string  `json:"id"`                  // 唯一标识
	Source    string  `json:"source"`              // 来源："gm", "event", "purchase", "quest", "system"
	Key       string  `json:"key"`                 // 属性键名，如 "productionBonus"
	Value     float64 `json:"value"`               // 数值
	Mode      string  `json:"mode"`                // "flat" | "percentAdd" | "percentMultiply"
	ExpiresAt string  `json:"expiresAt,omitempty"` // 到期时间（空 = 永久）
	CreatedAt string  `json:"createdAt"`           // 创建时间
	Note      string  `json:"note,omitempty"`      // GM 备注
}

// BuffListSource 通用 Buff 列表加成来源。
type BuffListSource struct {
	Buffs []Buff
}

// SourceName 返回 Buff 来源名称。
func (b *BuffListSource) SourceName() string { return "活动/GM" }

// ExpiresAt 返回 Buff 来源的过期时间列表。
func (b *BuffListSource) ExpiresAt() []time.Time {
	var times []time.Time
	for _, buff := range b.Buffs {
		if buff.ExpiresAt == "" {
			continue
		}
		if t, err := time.Parse(resourceDateLayout, buff.ExpiresAt); err == nil {
			times = append(times, t)
		}
	}
	return times
}

// Modifiers 返回当前仍生效的 Buff/DeBuff Modifier。
func (b *BuffListSource) Modifiers(now time.Time) []Modifier {
	var mods []Modifier
	for _, buff := range b.Buffs {
		if buff.ExpiresAt != "" {
			if t, err := time.Parse(resourceDateLayout, buff.ExpiresAt); err == nil && now.After(t) {
				continue
			}
		}
		mods = append(mods, Modifier{
			Key:   buff.Key,
			Value: buff.Value,
			Mode:  buff.Mode,
		})
	}
	return mods
}

// cleanExpiredBuffs 清理过期 Buff/DeBuff。
func cleanExpiredBuffs(state *GameState, now time.Time) {
	if len(state.Buffs) == 0 {
		return
	}
	remaining := state.Buffs[:0]
	for _, b := range state.Buffs {
		if b.ExpiresAt == "" {
			remaining = append(remaining, b)
			continue
		}
		if t, err := time.Parse(resourceDateLayout, b.ExpiresAt); err == nil && now.After(t) {
			continue
		}
		remaining = append(remaining, b)
	}
	state.Buffs = remaining
}

// validateBuffModifierSpec 校验 Buff/DeBuff 的 Modifier 范围。
func validateBuffModifierSpec(key string, mode string) error {
	if !IsValidStatKey(key) {
		return ErrInvalidBuffKey
	}
	if !IsValidModifierMode(mode) {
		return ErrInvalidBuffMode
	}
	return nil
}
