// 本文件定义将领被动特性使用的兵种固定属性 Modifier 键。
package game

import "strings"

const (
	unitAttackFlatModifierPrefix = "unitAttackFlat:"
	unitSpeedFlatModifierPrefix  = "unitSpeedFlat:"
)

// unitAttackFlatModifierKey 返回指定兵种固定攻击加成的隔离键。
func unitAttackFlatModifierKey(unitType string) string {
	return unitAttackFlatModifierPrefix + strings.TrimSpace(unitType)
}

// unitSpeedFlatModifierKey 返回指定兵种固定移动加成的隔离键。
func unitSpeedFlatModifierKey(unitType string) string {
	return unitSpeedFlatModifierPrefix + strings.TrimSpace(unitType)
}

// isUnitFlatModifierKey 判断 Modifier 是否为兵种固定属性。
func isUnitFlatModifierKey(key string) bool {
	return strings.HasPrefix(key, unitAttackFlatModifierPrefix) || strings.HasPrefix(key, unitSpeedFlatModifierPrefix)
}
