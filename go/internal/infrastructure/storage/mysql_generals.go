// 本文件归口 MySQL 玩家武将权威表和武将占用表同步。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"hero3/internal/app/game"
)

var errPlayerGeneralsMissing = errors.New("player_generals rows missing; run backfill-generals before using generals as authoritative state")

// overlayAuthoritativeGenerals 用 player_generals 和 player_general_assignments 覆盖兼容快照中的武将。
func (r *MySQLRepository) overlayAuthoritativeGenerals(state *game.GameState, playerID string) error {
	generals, found, err := loadPlayerGenerals(r.db, playerID)
	if err != nil {
		return err
	}
	assignments, _, err := loadPlayerGeneralAssignments(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeGenerals(state, generals, assignments, found)
}

// overlayAuthoritativeGeneralsTx 在事务内锁定并加载玩家武将权威状态。
func overlayAuthoritativeGeneralsTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	generals, found, err := loadPlayerGeneralsTx(tx, playerID)
	if err != nil {
		return err
	}
	assignments, _, err := loadPlayerGeneralAssignmentsTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeGenerals(state, generals, assignments, found)
}

// applyAuthoritativeGenerals 将武将权威表写回 GameState，旧单武将快照缺表时显式报错。
func applyAuthoritativeGenerals(state *game.GameState, generals []game.General, assignments []game.GeneralAssignment, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if state.General == nil && len(state.Generals) == 0 {
			state.Generals = []game.General{}
			state.GeneralAssignments = []game.GeneralAssignment{}
			return nil
		}
		return errPlayerGeneralsMissing
	}
	state.Generals = generals
	state.GeneralAssignments = assignments
	state.General = nil
	game.EnsureGeneralRoster(state, time.Now())
	return nil
}

func loadPlayerGenerals(queryer resourceQueryer, playerID string) ([]game.General, bool, error) {
	return loadPlayerGeneralsWithQuery(queryer, playerID, "")
}

func loadPlayerGeneralsTx(tx *sql.Tx, playerID string) ([]game.General, bool, error) {
	return loadPlayerGeneralsWithQuery(tx, playerID, " FOR UPDATE")
}

func loadPlayerGeneralsWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.General, bool, error) {
	rows, err := queryer.Query(
		`SELECT general_id, level, exp, stats_json
		 FROM player_generals
		 WHERE player_id = ?
		 ORDER BY general_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	generals := []game.General{}
	for rows.Next() {
		var general game.General
		var statsJSON []byte
		if err := rows.Scan(&general.ID, &general.Level, &general.Exp, &statsJSON); err != nil {
			return nil, false, err
		}
		general.ID = strings.TrimSpace(general.ID)
		if general.ID == "" {
			continue
		}
		if len(statsJSON) > 0 {
			_ = json.Unmarshal(statsJSON, &general.Stats)
		}
		if general.Stats == nil {
			general.Stats = map[string]int{}
		}
		hero, ok := game.GetHeroConfig(general.ID)
		if ok {
			general.Name = hero.Name
		} else {
			general.Name = general.ID
		}
		applyStorageGeneralConfig(&general)
		generals = append(generals, general)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return generals, len(generals) > 0, nil
}

// loadPlayerGeneralRowsTx 在事务内只锁定指定武将行。
func loadPlayerGeneralRowsTx(tx *sql.Tx, playerID string, generalIDs []string) ([]game.General, bool, error) {
	generalIDs = normalizeGeneralIDsForStorage(generalIDs)
	if len(generalIDs) == 0 {
		return []game.General{}, false, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(generalIDs)), ",")
	args := make([]any, 0, len(generalIDs)+1)
	args = append(args, playerID)
	for _, generalID := range generalIDs {
		args = append(args, generalID)
	}
	rows, err := tx.Query(
		`SELECT general_id, level, exp, stats_json
		 FROM player_generals
		 WHERE player_id = ? AND general_id IN (`+placeholders+`)
		 ORDER BY general_id
		 FOR UPDATE`,
		args...,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	generals := []game.General{}
	for rows.Next() {
		general, err := scanStorageGeneralRows(rows)
		if err != nil {
			return nil, false, err
		}
		generals = append(generals, general)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return generals, len(generals) > 0, nil
}

// scanStorageGeneralRows 把当前武将结果行还原为领域对象。
func scanStorageGeneralRows(rows *sql.Rows) (game.General, error) {
	var general game.General
	var statsJSON []byte
	if err := rows.Scan(&general.ID, &general.Level, &general.Exp, &statsJSON); err != nil {
		return game.General{}, err
	}
	general.ID = strings.TrimSpace(general.ID)
	if len(statsJSON) > 0 {
		_ = json.Unmarshal(statsJSON, &general.Stats)
	}
	if general.Stats == nil {
		general.Stats = map[string]int{}
	}
	if hero, ok := game.GetHeroConfig(general.ID); ok {
		general.Name = hero.Name
	} else {
		general.Name = general.ID
	}
	applyStorageGeneralConfig(&general)
	return general, nil
}

func loadPlayerGeneralAssignments(queryer resourceQueryer, playerID string) ([]game.GeneralAssignment, bool, error) {
	return loadPlayerGeneralAssignmentsWithQuery(queryer, playerID, "")
}

func loadPlayerGeneralAssignmentsTx(tx *sql.Tx, playerID string) ([]game.GeneralAssignment, bool, error) {
	return loadPlayerGeneralAssignmentsWithQuery(tx, playerID, " FOR UPDATE")
}

func loadPlayerGeneralAssignmentsWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.GeneralAssignment, bool, error) {
	rows, err := queryer.Query(
		`SELECT assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at
		 FROM player_general_assignments
		 WHERE player_id = ?
		 ORDER BY assignment_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	assignments := []game.GeneralAssignment{}
	for rows.Next() {
		var assignment game.GeneralAssignment
		var assignedAt sql.NullTime
		var endsAt sql.NullTime
		if err := rows.Scan(&assignment.ID, &assignment.GeneralID, &assignment.Slot, &assignment.ModuleID, &assignment.Status, &assignedAt, &endsAt); err != nil {
			return nil, false, err
		}
		assignment.ID = strings.TrimSpace(assignment.ID)
		assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
		if assignment.ID == "" || assignment.GeneralID == "" {
			continue
		}
		if assignedAt.Valid {
			assignment.AssignedAt = assignedAt.Time.UTC().Format(time.RFC3339)
		}
		if endsAt.Valid {
			assignment.EndsAt = endsAt.Time.UTC().Format(time.RFC3339)
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return assignments, len(assignments) > 0, nil
}

// loadPlayerGeneralAssignmentsForGeneralsTx 在事务内只锁定指定武将相关占用行。
func loadPlayerGeneralAssignmentsForGeneralsTx(tx *sql.Tx, playerID string, generalIDs []string) ([]game.GeneralAssignment, bool, error) {
	generalIDs = normalizeGeneralIDsForStorage(generalIDs)
	if len(generalIDs) == 0 {
		return []game.GeneralAssignment{}, false, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(generalIDs)), ",")
	args := make([]any, 0, len(generalIDs)+1)
	args = append(args, playerID)
	for _, generalID := range generalIDs {
		args = append(args, generalID)
	}
	rows, err := tx.Query(
		`SELECT assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at
		 FROM player_general_assignments
		 WHERE player_id = ? AND general_id IN (`+placeholders+`)
		 ORDER BY assignment_id
		 FOR UPDATE`,
		args...,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	assignments := []game.GeneralAssignment{}
	for rows.Next() {
		assignment, err := scanStorageGeneralAssignmentRows(rows)
		if err != nil {
			return nil, false, err
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return assignments, len(assignments) > 0, nil
}

// scanStorageGeneralAssignmentRows 把当前武将占用结果行还原为领域对象。
func scanStorageGeneralAssignmentRows(rows *sql.Rows) (game.GeneralAssignment, error) {
	var assignment game.GeneralAssignment
	var assignedAt sql.NullTime
	var endsAt sql.NullTime
	if err := rows.Scan(&assignment.ID, &assignment.GeneralID, &assignment.Slot, &assignment.ModuleID, &assignment.Status, &assignedAt, &endsAt); err != nil {
		return game.GeneralAssignment{}, err
	}
	assignment.ID = strings.TrimSpace(assignment.ID)
	assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
	if assignedAt.Valid {
		assignment.AssignedAt = assignedAt.Time.UTC().Format(time.RFC3339)
	}
	if endsAt.Valid {
		assignment.EndsAt = endsAt.Time.UTC().Format(time.RFC3339)
	}
	return assignment, nil
}

func syncPlayerGeneralsTx(tx *sql.Tx, playerID string, generals []game.General, updatedAt time.Time) error {
	generalIDs := generalIDsFromState(generals)
	if len(generalIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_generals WHERE player_id = ?`, playerID)
		return err
	}
	byID := generalsByID(generals)
	for _, generalID := range generalIDs {
		general := byID[generalID]
		statsJSON, err := json.Marshal(general.Stats)
		if err != nil {
			return err
		}
		hero, _ := game.GetHeroConfig(generalID)
		if _, err := tx.Exec(
			`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				faction = VALUES(faction),
				level = VALUES(level),
				exp = VALUES(exp),
				stats_json = VALUES(stats_json),
				updated_at = VALUES(updated_at)`,
			playerID,
			generalID,
			hero.Faction,
			general.Level,
			general.Exp,
			statsJSON,
			updatedAt.UTC(),
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerGeneralsTx(tx, playerID, generalIDs)
}

// normalizeGeneralIDsForStorage 清理武将 ID 列表。
func normalizeGeneralIDsForStorage(generalIDs []string) []string {
	idSet := map[string]struct{}{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" {
			continue
		}
		idSet[generalID] = struct{}{}
	}
	result := make([]string, 0, len(idSet))
	for generalID := range idSet {
		result = append(result, generalID)
	}
	sort.Strings(result)
	return result
}

func syncPlayerGeneralAssignmentsTx(tx *sql.Tx, playerID string, assignments []game.GeneralAssignment, updatedAt time.Time) error {
	assignmentIDs := generalAssignmentIDsFromState(assignments)
	if len(assignmentIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_general_assignments WHERE player_id = ?`, playerID)
		return err
	}
	byID := generalAssignmentsByID(assignments)
	for _, assignmentID := range assignmentIDs {
		assignment := byID[assignmentID]
		if _, err := tx.Exec(
			`INSERT INTO player_general_assignments (player_id, assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				general_id = VALUES(general_id),
				assignment_slot = VALUES(assignment_slot),
				module_id = VALUES(module_id),
				status = VALUES(status),
				assigned_at = VALUES(assigned_at),
				ends_at = VALUES(ends_at),
				updated_at = VALUES(updated_at)`,
			playerID,
			assignment.ID,
			assignment.GeneralID,
			assignment.Slot,
			assignment.ModuleID,
			assignment.Status,
			nullableTimeArg(parseStorageTime(assignment.AssignedAt)),
			nullableTimeArg(parseStorageTime(assignment.EndsAt)),
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerGeneralAssignmentsTx(tx, playerID, assignmentIDs)
}

type storageGeneralSnapshot struct {
	Level int
	Exp   int
	Stats string
}

type storageGeneralAssignmentSnapshot struct {
	GeneralID  string
	Slot       string
	ModuleID   string
	Status     string
	AssignedAt string
	EndsAt     string
}

func generalSnapshotChanged(before map[string]storageGeneralSnapshot, after []game.General) bool {
	return !generalSnapshotMapsEqual(before, generalSnapshotsFromStorageState(after))
}

func generalAssignmentSnapshotChanged(before map[string]storageGeneralAssignmentSnapshot, after []game.GeneralAssignment) bool {
	return !generalAssignmentSnapshotMapsEqual(before, generalAssignmentSnapshotsFromStorageState(after))
}

func generalSnapshotsFromStorageState(generals []game.General) map[string]storageGeneralSnapshot {
	result := map[string]storageGeneralSnapshot{}
	for _, general := range generals {
		general.ID = strings.TrimSpace(general.ID)
		if general.ID == "" {
			continue
		}
		statsJSON, _ := json.Marshal(general.Stats)
		result[general.ID] = storageGeneralSnapshot{Level: general.Level, Exp: general.Exp, Stats: string(statsJSON)}
	}
	return result
}

func generalAssignmentSnapshotsFromStorageState(assignments []game.GeneralAssignment) map[string]storageGeneralAssignmentSnapshot {
	result := map[string]storageGeneralAssignmentSnapshot{}
	for _, assignment := range assignments {
		assignment.ID = strings.TrimSpace(assignment.ID)
		assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
		if assignment.ID == "" || assignment.GeneralID == "" {
			continue
		}
		result[assignment.ID] = storageGeneralAssignmentSnapshot{
			GeneralID:  assignment.GeneralID,
			Slot:       assignment.Slot,
			ModuleID:   assignment.ModuleID,
			Status:     assignment.Status,
			AssignedAt: strings.TrimSpace(assignment.AssignedAt),
			EndsAt:     strings.TrimSpace(assignment.EndsAt),
		}
	}
	return result
}

func generalSnapshotMapsEqual(a map[string]storageGeneralSnapshot, b map[string]storageGeneralSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		if right, ok := b[key]; !ok || left != right {
			return false
		}
	}
	return true
}

func generalAssignmentSnapshotMapsEqual(a map[string]storageGeneralAssignmentSnapshot, b map[string]storageGeneralAssignmentSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		if right, ok := b[key]; !ok || left != right {
			return false
		}
	}
	return true
}

func generalIDsFromState(generals []game.General) []string {
	idSet := map[string]struct{}{}
	for _, general := range generals {
		generalID := strings.TrimSpace(general.ID)
		if generalID == "" {
			continue
		}
		idSet[generalID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func generalAssignmentIDsFromState(assignments []game.GeneralAssignment) []string {
	idSet := map[string]struct{}{}
	for _, assignment := range assignments {
		assignmentID := strings.TrimSpace(assignment.ID)
		if assignmentID == "" || strings.TrimSpace(assignment.GeneralID) == "" {
			continue
		}
		idSet[assignmentID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func generalsByID(generals []game.General) map[string]game.General {
	result := map[string]game.General{}
	for _, general := range generals {
		general.ID = strings.TrimSpace(general.ID)
		if general.ID == "" {
			continue
		}
		result[general.ID] = general
	}
	return result
}

func generalAssignmentsByID(assignments []game.GeneralAssignment) map[string]game.GeneralAssignment {
	result := map[string]game.GeneralAssignment{}
	for _, assignment := range assignments {
		assignment.ID = strings.TrimSpace(assignment.ID)
		if assignment.ID == "" || strings.TrimSpace(assignment.GeneralID) == "" {
			continue
		}
		result[assignment.ID] = assignment
	}
	return result
}

func deleteStalePlayerGeneralsTx(tx *sql.Tx, playerID string, generalIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(generalIDs)), ",")
	args := make([]any, 0, len(generalIDs)+1)
	args = append(args, playerID)
	for _, generalID := range generalIDs {
		args = append(args, generalID)
	}
	_, err := tx.Exec(`DELETE FROM player_generals WHERE player_id = ? AND general_id NOT IN (`+placeholders+`)`, args...)
	return err
}

func deleteStalePlayerGeneralAssignmentsTx(tx *sql.Tx, playerID string, assignmentIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(assignmentIDs)), ",")
	args := make([]any, 0, len(assignmentIDs)+1)
	args = append(args, playerID)
	for _, assignmentID := range assignmentIDs {
		args = append(args, assignmentID)
	}
	_, err := tx.Exec(`DELETE FROM player_general_assignments WHERE player_id = ? AND assignment_id NOT IN (`+placeholders+`)`, args...)
	return err
}

func applyStorageGeneralConfig(general *game.General) {
	if general == nil {
		return
	}
	// 通过公开的响应整理入口间接保持配置派生字段，避免存储层复制成长公式。
	state := game.GameState{General: general, Generals: []game.General{*general}}
	game.EnsureGeneralRoster(&state, time.Now())
	if state.General != nil {
		*general = *state.General
	}
}
