// 本文件实现内存仓储中的黄巾起义来袭队列。
package game

import (
	"errors"
	"sort"
	"time"
)

var ErrYellowTurbanMarchNotFound = errors.New("yellow turban march not found")

// CreateYellowTurbanMarch 创建一条黄巾来袭记录。
func (r *MemoryRepository) CreateYellowTurbanMarch(march YellowTurbanMarch) (YellowTurbanMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if march.ID == "" {
		march.ID = "yt_march_" + randomID(12)
	}
	r.yellowTurbanMarches[march.ID] = cloneYellowTurbanMarch(march)
	return march, nil
}

// GetYellowTurbanMarch 读取单条黄巾来袭记录。
func (r *MemoryRepository) GetYellowTurbanMarch(marchID string) (YellowTurbanMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	march, ok := r.yellowTurbanMarches[marchID]
	if !ok {
		return YellowTurbanMarch{}, ErrYellowTurbanMarchNotFound
	}
	return cloneYellowTurbanMarch(march), nil
}

// UpdateYellowTurbanMarch 更新单条黄巾来袭记录。
func (r *MemoryRepository) UpdateYellowTurbanMarch(marchID string, updatedAt time.Time, update func(march *YellowTurbanMarch) error) (YellowTurbanMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	march, ok := r.yellowTurbanMarches[marchID]
	if !ok {
		return YellowTurbanMarch{}, ErrYellowTurbanMarchNotFound
	}
	if update != nil {
		if err := update(&march); err != nil {
			return YellowTurbanMarch{}, err
		}
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	march.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.yellowTurbanMarches[marchID] = cloneYellowTurbanMarch(march)
	return march, nil
}

// ResolveYellowTurbanBattleTransaction 在内存事务中结算黄巾防守和驻防协防。
func (r *MemoryRepository) ResolveYellowTurbanBattleTransaction(marchID string, updatedAt time.Time, update func(defender *GameState, reinforcements []Reinforcement, march *YellowTurbanMarch) (BattleReport, []BattleReport, []Reinforcement, error)) (GameState, YellowTurbanMarch, BattleReport, []BattleReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	march, ok := r.yellowTurbanMarches[marchID]
	if !ok {
		return GameState{}, YellowTurbanMarch{}, BattleReport{}, nil, ErrYellowTurbanMarchNotFound
	}
	defender, ok := r.players[march.TargetPlayerID]
	if !ok {
		return GameState{}, YellowTurbanMarch{}, BattleReport{}, nil, ErrPlayerNotFound
	}
	targetRecords := []Reinforcement{}
	for _, record := range r.reinforcements {
		normalizeGarrisonRecord(&record)
		if record.HostPlayerID == defender.Player.ID && record.Status == ReinforcementStatusStationed && record.Rules.CanFight {
			targetRecords = append(targetRecords, cloneReinforcement(record))
		}
	}
	report, reinforcementReports, changedReinforcements, err := update(&defender, targetRecords, &march)
	if err != nil {
		return GameState{}, YellowTurbanMarch{}, BattleReport{}, nil, err
	}
	for _, record := range changedReinforcements {
		r.reinforcements[record.ID] = record
	}
	r.players[defender.Player.ID] = defender
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	r.playerUpdatedAt[defender.Player.ID] = updatedAt.UTC()
	march.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.yellowTurbanMarches[march.ID] = cloneYellowTurbanMarch(march)
	return defender, cloneYellowTurbanMarch(march), report, reinforcementReports, nil
}

// ListYellowTurbanMarchesForPlayer 返回玩家相关黄巾来袭。
func (r *MemoryRepository) ListYellowTurbanMarchesForPlayer(playerID string) ([]YellowTurbanMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []YellowTurbanMarch{}
	for _, march := range r.yellowTurbanMarches {
		if march.TargetPlayerID == playerID {
			items = append(items, cloneYellowTurbanMarch(march))
		}
	}
	sortYellowTurbanMarches(items)
	return items, nil
}

// CountActiveYellowTurbanMarches 统计玩家未结束的黄巾来袭。
func (r *MemoryRepository) CountActiveYellowTurbanMarches(playerID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, march := range r.yellowTurbanMarches {
		if march.TargetPlayerID == playerID && yellowTurbanMarchActive(march.Status) {
			count++
		}
	}
	return count, nil
}

// ListDueYellowTurbanMarches 返回已经到达、待结算的黄巾来袭。
func (r *MemoryRepository) ListDueYellowTurbanMarches(playerID string, now time.Time) ([]YellowTurbanMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []YellowTurbanMarch{}
	for _, march := range r.yellowTurbanMarches {
		if playerID != "" && march.TargetPlayerID != playerID {
			continue
		}
		if march.Status != YellowTurbanMarchStatusMarching {
			continue
		}
		arrivesAt, err := time.Parse(resourceDateLayout, march.ArrivesAt)
		if err != nil || arrivesAt.After(now) {
			continue
		}
		items = append(items, cloneYellowTurbanMarch(march))
	}
	sortYellowTurbanMarches(items)
	return items, nil
}

// cloneYellowTurbanMarch 复制黄巾来袭记录。
func cloneYellowTurbanMarch(march YellowTurbanMarch) YellowTurbanMarch {
	march.Troops = cloneStringIntMap(march.Troops)
	return march
}

// sortYellowTurbanMarches 稳定排序黄巾来袭。
func sortYellowTurbanMarches(items []YellowTurbanMarch) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ArrivesAt != items[j].ArrivesAt {
			return items[i].ArrivesAt < items[j].ArrivesAt
		}
		return items[i].ID < items[j].ID
	})
}

// yellowTurbanMarchActive 判断状态是否占用来袭路数。
func yellowTurbanMarchActive(status string) bool {
	return status == YellowTurbanMarchStatusMarching || status == YellowTurbanMarchStatusResolving
}
