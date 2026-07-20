// 本文件统一战后返兵按兵种 ID 的稳定顺序，避免总上限触发时受 Go map 遍历顺序影响。
package traits

import "sort"

// sortedLossUnitTypes 返回有损失兵种的稳定升序列表。
func sortedLossUnitTypes(losses map[string]int) []string {
	unitTypes := make([]string, 0, len(losses))
	for unitType, lost := range losses {
		if lost > 0 {
			unitTypes = append(unitTypes, unitType)
		}
	}
	sort.Strings(unitTypes)
	return unitTypes
}
