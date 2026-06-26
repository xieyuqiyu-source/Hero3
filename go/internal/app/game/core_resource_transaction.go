// 本文件归口核心资源事务内的数量、容量和扣减规则。
package game

// ensureResourceState 确保资源数量和容量 map 已初始化。
func ensureResourceState(state *GameState) error {
	if state == nil {
		return ErrPlayerNotFound
	}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}
	if state.Resources.Capacity == nil {
		state.Resources.Capacity = map[string]int{}
	}
	return nil
}

// spendResources 在事务回调内统一扣减资源，任一资源不足则整体失败。
func spendResources(state *GameState, costs map[string]int) error {
	if err := ensureResourceState(state); err != nil {
		return err
	}
	for resourceType, cost := range costs {
		if cost <= 0 {
			continue
		}
		if state.Resources.Items[resourceType] < cost {
			return ErrInsufficientRes
		}
	}
	for resourceType, cost := range costs {
		if cost <= 0 {
			continue
		}
		state.Resources.Items[resourceType] -= cost
		if state.Resources.Items[resourceType] < 0 {
			state.Resources.Items[resourceType] = 0
		}
	}
	return nil
}

// canSpendResources 判断当前事务内资源是否足够。
func canSpendResources(state *GameState, costs map[string]int) bool {
	if err := ensureResourceState(state); err != nil {
		return false
	}
	for resourceType, cost := range costs {
		if cost <= 0 {
			continue
		}
		if state.Resources.Items[resourceType] < cost {
			return false
		}
	}
	return true
}

// addResourceCapped 在事务回调内按容量上限发放单项资源。
func addResourceCapped(state *GameState, resourceType string, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, nil
	}
	if err := ensureResourceState(state); err != nil {
		return 0, 0, err
	}
	current := state.Resources.Items[resourceType]
	next := current + amount
	overflow := 0
	if capacity := state.Resources.Capacity[resourceType]; capacity > 0 && next > capacity {
		overflow = next - capacity
		next = capacity
	}
	state.Resources.Items[resourceType] = next
	return next - current, overflow, nil
}

// adjustResourceCapped 在事务回调内统一调整资源，负数不会扣到 0 以下，正数受容量限制。
func adjustResourceCapped(state *GameState, resourceType string, delta int) (int, error) {
	if err := ensureResourceState(state); err != nil {
		return 0, err
	}
	current := state.Resources.Items[resourceType]
	next := current + delta
	if next < 0 {
		next = 0
	}
	if capacity := state.Resources.Capacity[resourceType]; capacity > 0 && next > capacity {
		next = capacity
	}
	state.Resources.Items[resourceType] = next
	return next - current, nil
}

// fillResourcesToCapacity 在事务回调内把所有已有容量资源补满。
func fillResourcesToCapacity(state *GameState) error {
	if err := ensureResourceState(state); err != nil {
		return err
	}
	for resourceType, capacity := range state.Resources.Capacity {
		if capacity < 0 {
			capacity = 0
		}
		state.Resources.Items[resourceType] = capacity
	}
	return nil
}

// replaceResourceItems 在事务回调内替换资源数量快照，用于自然结算后的统一写回。
func replaceResourceItems(state *GameState, items map[string]int) error {
	if err := ensureResourceState(state); err != nil {
		return err
	}
	state.Resources.Items = copyResourceMap(items)
	return nil
}

// replaceResourceCapacity 在事务回调内替换资源容量快照，用于建筑和加成结算后的统一写回。
func replaceResourceCapacity(state *GameState, capacity map[string]int) error {
	if err := ensureResourceState(state); err != nil {
		return err
	}
	state.Resources.Capacity = copyResourceMap(capacity)
	return nil
}
