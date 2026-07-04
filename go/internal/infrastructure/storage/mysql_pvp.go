// 本文件归口 MySQL PVP 仓储方法，负责行军、战斗和跨玩家结算落库。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hero3/internal/app/game"
)

const pvpMarchColumns = `id, attacker_player_id, attacker_name, attacker_faction,
	defender_player_id, defender_name, defender_faction, march_type, status,
	attack_troops_json, attack_generals_json, speed_multiplier, duration_seconds,
	started_at, arrives_at, return_started_at, returns_at, resolved_at,
	attacker_report_id, defender_report_id, battle_id, accelerated_times, created_at, updated_at`

const pvpBattleColumns = `id, march_id, attacker_player_id, defender_player_id, status,
	attacker_snapshot_json, defender_snapshot_json, reinforcement_snapshot_json,
	result_json, losses_json, plunder_json, attacker_report_id, defender_report_id,
	resolved_at, created_at, updated_at`

const pvpPlayerStateColumns = `player_id, status, protection_type, protected_until, cooldown_until,
	daily_attack_count, daily_attack_limit, daily_reset_at, target_cooldown_json,
	metadata_json, created_at, updated_at`

// CreatePvpMarchWithState 创建 PVP 行军，并在同一事务内扣出攻击方兵力和占用武将。
func (r *MySQLRepository) CreatePvpMarchWithState(attackerPlayerID string, defenderPlayerID string, updatedAt time.Time, update func(attacker *game.GameState, defender *game.GameState) (game.PvpMarch, error)) (game.GameState, game.GameState, game.PvpMarch, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	defer tx.Rollback()
	attacker, attackerJSON, defender, defenderJSON, err := loadPvpPlayerPairOrderedTx(tx, attackerPlayerID, defenderPlayerID)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	attackerArmy := armySnapshotsFromStorageState(attacker.Army)
	attackerAssignments := generalAssignmentSnapshotsFromStorageState(attacker.GeneralAssignments)
	defenderArmy := armySnapshotsFromStorageState(defender.Army)
	defenderAssignments := generalAssignmentSnapshotsFromStorageState(defender.GeneralAssignments)
	march, err := update(&attacker, &defender)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	if err := savePvpPlayerStateTx(tx, attacker.Player.ID, attacker, attackerJSON, updatedAt, attackerArmy, attackerAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	if err := savePvpPlayerStateTx(tx, defender.Player.ID, defender, defenderJSON, updatedAt, defenderArmy, defenderAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	if err := insertPvpMarchTx(tx, march); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, err
	}
	return attacker, defender, march, nil
}

// UpdatePvpScoutStates 在同一事务内更新 PVP 侦查双方状态。
func (r *MySQLRepository) UpdatePvpScoutStates(scoutPlayerID string, targetPlayerID string, updatedAt time.Time, update func(scout *game.GameState, target *game.GameState) error) (game.GameState, game.GameState, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	defer tx.Rollback()
	scout, scoutJSON, target, targetJSON, err := loadPvpPlayerPairOrderedTx(tx, scoutPlayerID, targetPlayerID)
	if err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	scoutArmy := armySnapshotsFromStorageState(scout.Army)
	scoutAssignments := generalAssignmentSnapshotsFromStorageState(scout.GeneralAssignments)
	targetArmy := armySnapshotsFromStorageState(target.Army)
	targetAssignments := generalAssignmentSnapshotsFromStorageState(target.GeneralAssignments)
	if err := update(&scout, &target); err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	if err := savePvpPlayerStateTx(tx, scout.Player.ID, scout, scoutJSON, updatedAt, scoutArmy, scoutAssignments); err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	if err := savePvpPlayerStateTx(tx, target.Player.ID, target, targetJSON, updatedAt, targetArmy, targetAssignments); err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.GameState{}, err
	}
	return scout, target, nil
}

// GetPvpMarch 读取单条 PVP 行军。
func (r *MySQLRepository) GetPvpMarch(marchID string) (game.PvpMarch, error) {
	return getPvpMarchTx(r.db, marchID, "")
}

// UpdatePvpMarch 更新单条 PVP 行军。
func (r *MySQLRepository) UpdatePvpMarch(marchID string, updatedAt time.Time, update func(march *game.PvpMarch) error) (game.PvpMarch, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.PvpMarch{}, err
	}
	defer tx.Rollback()
	march, err := getPvpMarchTx(tx, marchID, " FOR UPDATE")
	if err != nil {
		return game.PvpMarch{}, err
	}
	if err := update(&march); err != nil {
		return game.PvpMarch{}, err
	}
	march.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if err := updatePvpMarchTx(tx, march); err != nil {
		return game.PvpMarch{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.PvpMarch{}, err
	}
	return march, nil
}

// UpdatePvpMarchWithAttackerState 同时更新 PVP 行军和攻击方状态。
func (r *MySQLRepository) UpdatePvpMarchWithAttackerState(marchID string, updatedAt time.Time, update func(attacker *game.GameState, march *game.PvpMarch) error) (game.GameState, game.PvpMarch, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	defer tx.Rollback()
	march, err := getPvpMarchTx(tx, marchID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	attacker, attackerJSON, err := loadPvpPlayerStateTx(tx, march.AttackerPlayerID)
	if err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	previousArmy := armySnapshotsFromStorageState(attacker.Army)
	previousAssignments := generalAssignmentSnapshotsFromStorageState(attacker.GeneralAssignments)
	if err := update(&attacker, &march); err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	attacker.ServerTime = updatedAt.UTC().Format(time.RFC3339)
	march.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if err := savePvpPlayerStateTx(tx, attacker.Player.ID, attacker, attackerJSON, updatedAt, previousArmy, previousAssignments); err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	if err := updatePvpMarchTx(tx, march); err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.PvpMarch{}, err
	}
	return attacker, march, nil
}

// ListPvpMarchesForPlayer 返回玩家相关 PVP 行军。
func (r *MySQLRepository) ListPvpMarchesForPlayer(playerID string) ([]game.PvpMarch, error) {
	rows, err := r.db.Query(
		`SELECT `+pvpMarchColumns+`
		 FROM pvp_marches
		 WHERE attacker_player_id = ? OR defender_player_id = ?
		 ORDER BY updated_at DESC
		 LIMIT 100`,
		playerID, playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPvpMarchRows(rows)
}

// ListDuePvpMarches 返回已经到达且待结算的 PVP 行军。
func (r *MySQLRepository) ListDuePvpMarches(playerID string, now time.Time) ([]game.PvpMarch, error) {
	rows, err := r.db.Query(
		`SELECT `+pvpMarchColumns+`
		 FROM pvp_marches
		 WHERE ((status = ? AND arrives_at <= ?) OR (status = ? AND returns_at <= ?))
		   AND (attacker_player_id = ? OR defender_player_id = ?)
		 ORDER BY arrives_at ASC
		 LIMIT 20`,
		game.PvpMarchStatusMarching, now.UTC(), game.PvpMarchStatusReturning, now.UTC(), playerID, playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPvpMarchRows(rows)
}

// ResolvePvpBattleTransaction 在事务内抢占行军并完成 PVP 战斗结算。
func (r *MySQLRepository) ResolvePvpBattleTransaction(marchID string, updatedAt time.Time, update func(attacker *game.GameState, defender *game.GameState, reinforcements []game.Reinforcement, march *game.PvpMarch) (game.PvpBattle, game.BattleReport, game.BattleReport, []game.BattleReport, []game.Reinforcement, error)) (game.GameState, game.GameState, game.PvpMarch, game.PvpBattle, game.BattleReport, game.BattleReport, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	defer tx.Rollback()
	march, err := getPvpMarchTx(tx, marchID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	if march.Status == game.PvpMarchStatusResolved {
		battle, _ := getPvpBattleByMarchTx(tx, march.ID)
		attacker, _, defender, _, _ := loadPvpPlayerPairOrderedTx(tx, march.AttackerPlayerID, march.DefenderPlayerID)
		return attacker, defender, march, battle, game.BattleReport{}, game.BattleReport{}, nil
	}
	if march.Status != game.PvpMarchStatusMarching && march.Status != game.PvpMarchStatusResolving {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, game.ErrInvalidReinforcement
	}
	attacker, attackerJSON, defender, defenderJSON, err := loadPvpPlayerPairOrderedTx(tx, march.AttackerPlayerID, march.DefenderPlayerID)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	attackerArmy := armySnapshotsFromStorageState(attacker.Army)
	attackerAssignments := generalAssignmentSnapshotsFromStorageState(attacker.GeneralAssignments)
	defenderArmy := armySnapshotsFromStorageState(defender.Army)
	defenderAssignments := generalAssignmentSnapshotsFromStorageState(defender.GeneralAssignments)
	reinforcements, err := listReceivedReinforcementsTx(tx, defender.Player.ID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	battle, attackerReport, defenderReport, reinforcementReports, changedReinforcements, err := update(&attacker, &defender, reinforcements, &march)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	if err := savePvpPlayerStateTx(tx, attacker.Player.ID, attacker, attackerJSON, updatedAt, attackerArmy, attackerAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	if err := savePvpPlayerStateTx(tx, defender.Player.ID, defender, defenderJSON, updatedAt, defenderArmy, defenderAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	for _, record := range changedReinforcements {
		if err := updateReinforcementTx(tx, record.ID, record); err != nil {
			return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
		}
	}
	if battle.ID != "" {
		if err := insertPvpBattleTx(tx, battle); err != nil && !isDuplicateEntry(err) {
			return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
		}
	}
	if err := updatePvpMarchTx(tx, march); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	if attackerReport.ID != "" {
		if err := insertBattleReportTx(tx, attackerReport); err != nil && !isDuplicateEntry(err) {
			return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
		}
	}
	if defenderReport.ID != "" {
		if err := insertBattleReportTx(tx, defenderReport); err != nil && !isDuplicateEntry(err) {
			return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
		}
	}
	for _, report := range reinforcementReports {
		if report.ID == "" {
			continue
		}
		if err := insertBattleReportTx(tx, report); err != nil && !isDuplicateEntry(err) {
			return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.GameState{}, game.PvpMarch{}, game.PvpBattle{}, game.BattleReport{}, game.BattleReport{}, err
	}
	return attacker, defender, march, battle, attackerReport, defenderReport, nil
}

// GetPvpBattle 读取单条 PVP 战斗。
func (r *MySQLRepository) GetPvpBattle(battleID string) (game.PvpBattle, error) {
	return getPvpBattleTx(r.db, battleID)
}

// ListPvpBattlesForPlayer 返回玩家相关 PVP 战斗。
func (r *MySQLRepository) ListPvpBattlesForPlayer(playerID string) ([]game.PvpBattle, error) {
	rows, err := r.db.Query(
		`SELECT `+pvpBattleColumns+`
		 FROM pvp_battles
		 WHERE attacker_player_id = ? OR defender_player_id = ?
		 ORDER BY updated_at DESC
		 LIMIT 100`,
		playerID, playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPvpBattleRows(rows)
}

// GetPvpPlayerState 读取或初始化玩家 PVP 状态。
func (r *MySQLRepository) GetPvpPlayerState(playerID string, now time.Time) (game.PvpPlayerState, error) {
	state, err := scanPvpPlayerState(r.db.QueryRow(
		`SELECT `+pvpPlayerStateColumns+` FROM pvp_player_states WHERE player_id = ? LIMIT 1`,
		playerID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := r.GetState(playerID); err != nil {
			return game.PvpPlayerState{}, err
		}
		state = game.PvpPlayerState{
			PlayerID: playerID,
		}
		state = game.NewDefaultPvpPlayerStateForStorage(state.PlayerID, now)
		if err := r.SavePvpPlayerState(state, now); err != nil {
			return game.PvpPlayerState{}, err
		}
		return state, nil
	}
	if err != nil {
		return game.PvpPlayerState{}, err
	}
	state = game.NormalizePvpPlayerStateForStorage(state, now)
	if err := r.SavePvpPlayerState(state, now); err != nil {
		return game.PvpPlayerState{}, err
	}
	return state, nil
}

// SavePvpPlayerState 保存玩家 PVP 状态。
func (r *MySQLRepository) SavePvpPlayerState(state game.PvpPlayerState, updatedAt time.Time) error {
	targetCooldownJSON, _ := json.Marshal(state.TargetCooldown)
	metadataJSON, _ := json.Marshal(state.Metadata)
	_, err := r.db.Exec(
		`INSERT INTO pvp_player_states (
			player_id, status, protection_type, protected_until, cooldown_until,
			daily_attack_count, daily_attack_limit, daily_reset_at, target_cooldown_json,
			metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			protection_type = VALUES(protection_type),
			protected_until = VALUES(protected_until),
			cooldown_until = VALUES(cooldown_until),
			daily_attack_count = VALUES(daily_attack_count),
			daily_attack_limit = VALUES(daily_attack_limit),
			daily_reset_at = VALUES(daily_reset_at),
			target_cooldown_json = VALUES(target_cooldown_json),
			metadata_json = VALUES(metadata_json),
			updated_at = VALUES(updated_at)`,
		state.PlayerID, state.Status, state.ProtectionType, parseNullableTime(state.ProtectedUntil), parseNullableTime(state.CooldownUntil),
		state.DailyAttackCount, state.DailyAttackLimit, parseNullableTime(state.DailyResetAt), nullableJSON(targetCooldownJSON),
		nullableJSON(metadataJSON), parseNullableTime(state.CreatedAt), updatedAt.UTC(),
	)
	return err
}

// GetCurrentPvpSeason 读取当前时间所在的激活 PVP 赛季。
func (r *MySQLRepository) GetCurrentPvpSeason(now time.Time) (game.PvpSeasonRecord, error) {
	return scanPvpSeason(r.db.QueryRow(
		`SELECT id, name, status, starts_at, ends_at, settled_at, rules_json, rewards_json, created_at, updated_at
		 FROM pvp_seasons
		 WHERE status = ? AND starts_at <= ? AND ends_at > ?
		 ORDER BY starts_at DESC
		 LIMIT 1`,
		game.PvpSeasonStatusActive, now.UTC(), now.UTC(),
	))
}

// SavePvpSeason 保存 PVP 赛季定义。
func (r *MySQLRepository) SavePvpSeason(season game.PvpSeasonRecord, updatedAt time.Time) error {
	rulesJSON, _ := json.Marshal(season.Rules)
	rewardsJSON, _ := json.Marshal(season.Rewards)
	_, err := r.db.Exec(
		`INSERT INTO pvp_seasons (
			id, name, status, starts_at, ends_at, settled_at, rules_json, rewards_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			status = VALUES(status),
			starts_at = VALUES(starts_at),
			ends_at = VALUES(ends_at),
			settled_at = VALUES(settled_at),
			rules_json = VALUES(rules_json),
			rewards_json = VALUES(rewards_json),
			updated_at = VALUES(updated_at)`,
		season.ID, season.Name, season.Status, parseNullableTime(season.StartsAt), parseNullableTime(season.EndsAt), parseNullableTime(season.SettledAt),
		nullableJSON(rulesJSON), nullableJSON(rewardsJSON), parseNullableTime(season.CreatedAt), updatedAt.UTC(),
	)
	return err
}

// ListPvpSeasons 返回全部 PVP 赛季。
func (r *MySQLRepository) ListPvpSeasons() ([]game.PvpSeasonRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, name, status, starts_at, ends_at, settled_at, rules_json, rewards_json, created_at, updated_at
		 FROM pvp_seasons
		 ORDER BY starts_at DESC
		 LIMIT 100`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []game.PvpSeasonRecord{}
	for rows.Next() {
		item, err := scanPvpSeason(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SavePvpSeasonPlayers 保存赛季玩家结算快照。
func (r *MySQLRepository) SavePvpSeasonPlayers(seasonID string, players []game.PvpSeasonPlayerRecord, updatedAt time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	updatedAtText := updatedAt.UTC().Format(time.RFC3339)
	for _, player := range players {
		if player.CreatedAt == "" {
			player.CreatedAt = updatedAtText
		}
		_, err := tx.Exec(
			`INSERT INTO pvp_season_players (
				season_id, player_id, nickname, faction, `+"`rank`"+`, points, rating, wins, losses,
				defense_wins, defense_losses, last_battle_at, reward_mail_id, reward_sent_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				nickname = VALUES(nickname),
				faction = VALUES(faction),
				`+"`rank`"+` = VALUES(`+"`rank`"+`),
				points = VALUES(points),
				rating = VALUES(rating),
				wins = VALUES(wins),
				losses = VALUES(losses),
				defense_wins = VALUES(defense_wins),
				defense_losses = VALUES(defense_losses),
				last_battle_at = VALUES(last_battle_at),
				reward_mail_id = VALUES(reward_mail_id),
				reward_sent_at = VALUES(reward_sent_at),
				updated_at = VALUES(updated_at)`,
			seasonID, player.PlayerID, player.Nickname, player.Faction, player.Rank, player.Points, player.Rating, player.Wins, player.Losses,
			player.DefenseWins, player.DefenseLosses, parseNullableTime(player.LastBattleAt), player.RewardMailID, parseNullableTime(player.RewardSentAt),
			parseNullableTime(player.CreatedAt), updatedAt.UTC(),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListPvpSeasonPlayers 返回赛季玩家结算快照。
func (r *MySQLRepository) ListPvpSeasonPlayers(seasonID string) ([]game.PvpSeasonPlayerRecord, error) {
	rows, err := r.db.Query(
		`SELECT season_id, player_id, nickname, faction, `+"`rank`"+`, points, rating, wins, losses,
			defense_wins, defense_losses, last_battle_at, reward_mail_id, reward_sent_at, created_at, updated_at
		 FROM pvp_season_players
		 WHERE season_id = ?
		 ORDER BY `+"`rank`"+` ASC`,
		seasonID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []game.PvpSeasonPlayerRecord{}
	for rows.Next() {
		var item game.PvpSeasonPlayerRecord
		var lastBattleAt, rewardSentAt, createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&item.SeasonID, &item.PlayerID, &item.Nickname, &item.Faction, &item.Rank, &item.Points, &item.Rating, &item.Wins, &item.Losses,
			&item.DefenseWins, &item.DefenseLosses, &lastBattleAt, &item.RewardMailID, &rewardSentAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		item.LastBattleAt = formatNullTime(lastBattleAt)
		item.RewardSentAt = formatNullTime(rewardSentAt)
		item.CreatedAt = formatNullTime(createdAt)
		item.UpdatedAt = formatNullTime(updatedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

// scanPvpSeason 扫描一条 PVP 赛季记录。
func scanPvpSeason(scanner reinforcementScanner) (game.PvpSeasonRecord, error) {
	var season game.PvpSeasonRecord
	var startsAt, endsAt, settledAt, createdAt, updatedAt sql.NullTime
	var rulesJSON, rewardsJSON sql.NullString
	err := scanner.Scan(
		&season.ID, &season.Name, &season.Status, &startsAt, &endsAt, &settledAt,
		&rulesJSON, &rewardsJSON, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return game.PvpSeasonRecord{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.PvpSeasonRecord{}, err
	}
	season.StartsAt = formatNullTime(startsAt)
	season.EndsAt = formatNullTime(endsAt)
	season.SettledAt = formatNullTime(settledAt)
	season.CreatedAt = formatNullTime(createdAt)
	season.UpdatedAt = formatNullTime(updatedAt)
	unmarshalNullJSON(rulesJSON, &season.Rules)
	unmarshalNullJSON(rewardsJSON, &season.Rewards)
	return season, nil
}

func loadPvpPlayerStateTx(tx *sql.Tx, playerID string) (game.GameState, []byte, error) {
	state, previousJSON, err := loadReinforcementPlayerStateTx(tx, playerID)
	if err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, time.Now()); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeBuildingsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeResourceSlotsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeRecruitQueuesTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	return state, previousJSON, nil
}

func savePvpPlayerStateTx(tx *sql.Tx, playerID string, state game.GameState, previousJSON []byte, updatedAt time.Time, previousArmy map[string]storageArmySnapshot, previousAssignments map[string]storageGeneralAssignmentSnapshot) error {
	if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
		return err
	}
	if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
		return err
	}
	if err := saveReinforcementPlayerStateTx(tx, playerID, state, previousJSON, updatedAt, previousArmy, previousAssignments, currencySnapshotFromState(state)); err != nil {
		return err
	}
	return nil
}

func getPvpMarchTx(queryer reinforcementQueryer, marchID string, lockClause string) (game.PvpMarch, error) {
	return scanPvpMarch(queryer.QueryRow(
		`SELECT `+pvpMarchColumns+`
		 FROM pvp_marches
		 WHERE id = ?
		 LIMIT 1`+lockClause,
		marchID,
	))
}

func insertPvpMarchTx(tx *sql.Tx, march game.PvpMarch) error {
	troopsJSON, _ := json.Marshal(march.AttackTroops)
	generalsJSON, _ := json.Marshal(march.AttackGenerals)
	_, err := tx.Exec(
		`INSERT INTO pvp_marches (
			id, attacker_player_id, attacker_name, attacker_faction, defender_player_id, defender_name, defender_faction,
			march_type, status, attack_troops_json, attack_generals_json, speed_multiplier, duration_seconds,
			started_at, arrives_at, return_started_at, returns_at, resolved_at,
			attacker_report_id, defender_report_id, battle_id, accelerated_times, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		march.ID, march.AttackerPlayerID, march.AttackerName, march.AttackerFaction, march.DefenderPlayerID, march.DefenderName, march.DefenderFaction,
		march.MarchType, march.Status, troopsJSON, nullableJSON(generalsJSON), march.SpeedMultiplier, march.DurationSeconds,
		parseNullableTime(march.StartedAt), parseNullableTime(march.ArrivesAt), parseNullableTime(march.ReturnStartedAt), parseNullableTime(march.ReturnsAt), parseNullableTime(march.ResolvedAt),
		march.AttackerReportID, march.DefenderReportID, march.BattleID, march.AcceleratedTimes, parseNullableTime(march.CreatedAt), parseNullableTime(march.UpdatedAt),
	)
	return err
}

func updatePvpMarchTx(tx *sql.Tx, march game.PvpMarch) error {
	troopsJSON, _ := json.Marshal(march.AttackTroops)
	generalsJSON, _ := json.Marshal(march.AttackGenerals)
	_, err := tx.Exec(
		`UPDATE pvp_marches SET
			attacker_player_id = ?, attacker_name = ?, attacker_faction = ?, defender_player_id = ?, defender_name = ?, defender_faction = ?,
			march_type = ?, status = ?, attack_troops_json = ?, attack_generals_json = ?, speed_multiplier = ?, duration_seconds = ?,
			started_at = ?, arrives_at = ?, return_started_at = ?, returns_at = ?, resolved_at = ?,
			attacker_report_id = ?, defender_report_id = ?, battle_id = ?, accelerated_times = ?, created_at = ?, updated_at = ?
		 WHERE id = ?`,
		march.AttackerPlayerID, march.AttackerName, march.AttackerFaction, march.DefenderPlayerID, march.DefenderName, march.DefenderFaction,
		march.MarchType, march.Status, troopsJSON, nullableJSON(generalsJSON), march.SpeedMultiplier, march.DurationSeconds,
		parseNullableTime(march.StartedAt), parseNullableTime(march.ArrivesAt), parseNullableTime(march.ReturnStartedAt), parseNullableTime(march.ReturnsAt), parseNullableTime(march.ResolvedAt),
		march.AttackerReportID, march.DefenderReportID, march.BattleID, march.AcceleratedTimes, parseNullableTime(march.CreatedAt), parseNullableTime(march.UpdatedAt), march.ID,
	)
	return err
}

func scanPvpMarchRows(rows *sql.Rows) ([]game.PvpMarch, error) {
	items := []game.PvpMarch{}
	for rows.Next() {
		item, err := scanPvpMarch(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPvpMarch(scanner reinforcementScanner) (game.PvpMarch, error) {
	var march game.PvpMarch
	var troopsJSON []byte
	var generalsJSON sql.NullString
	var startedAt, arrivesAt, returnStartedAt, returnsAt, resolvedAt, createdAt, updatedAt sql.NullTime
	err := scanner.Scan(
		&march.ID, &march.AttackerPlayerID, &march.AttackerName, &march.AttackerFaction,
		&march.DefenderPlayerID, &march.DefenderName, &march.DefenderFaction, &march.MarchType, &march.Status,
		&troopsJSON, &generalsJSON, &march.SpeedMultiplier, &march.DurationSeconds,
		&startedAt, &arrivesAt, &returnStartedAt, &returnsAt, &resolvedAt,
		&march.AttackerReportID, &march.DefenderReportID, &march.BattleID, &march.AcceleratedTimes, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return game.PvpMarch{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.PvpMarch{}, err
	}
	_ = json.Unmarshal(troopsJSON, &march.AttackTroops)
	unmarshalNullJSON(generalsJSON, &march.AttackGenerals)
	march.StartedAt = formatNullTime(startedAt)
	march.ArrivesAt = formatNullTime(arrivesAt)
	march.ReturnStartedAt = formatNullTime(returnStartedAt)
	march.ReturnsAt = formatNullTime(returnsAt)
	march.ResolvedAt = formatNullTime(resolvedAt)
	march.CreatedAt = formatNullTime(createdAt)
	march.UpdatedAt = formatNullTime(updatedAt)
	return march, nil
}

func insertPvpBattleTx(tx *sql.Tx, battle game.PvpBattle) error {
	attackerSnapshotJSON, _ := json.Marshal(battle.AttackerSnapshot)
	defenderSnapshotJSON, _ := json.Marshal(battle.DefenderSnapshot)
	reinforcementSnapshotJSON, _ := json.Marshal(battle.ReinforcementSnapshot)
	resultJSON, _ := json.Marshal(battle.Result)
	lossesJSON, _ := json.Marshal(battle.Losses)
	plunderJSON, _ := json.Marshal(battle.Plunder)
	_, err := tx.Exec(
		`INSERT INTO pvp_battles (
			id, march_id, attacker_player_id, defender_player_id, status,
			attacker_snapshot_json, defender_snapshot_json, reinforcement_snapshot_json,
			result_json, losses_json, plunder_json, attacker_report_id, defender_report_id,
			resolved_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		battle.ID, battle.MarchID, battle.AttackerPlayerID, battle.DefenderPlayerID, battle.Status,
		nullableJSON(attackerSnapshotJSON), nullableJSON(defenderSnapshotJSON), nullableJSON(reinforcementSnapshotJSON),
		nullableJSON(resultJSON), nullableJSON(lossesJSON), nullableJSON(plunderJSON), battle.AttackerReportID, battle.DefenderReportID,
		parseNullableTime(battle.ResolvedAt), parseNullableTime(battle.CreatedAt), parseNullableTime(battle.UpdatedAt),
	)
	return err
}

func getPvpBattleTx(queryer reinforcementQueryer, battleID string) (game.PvpBattle, error) {
	return scanPvpBattle(queryer.QueryRow(
		`SELECT `+pvpBattleColumns+`
		 FROM pvp_battles
		 WHERE id = ?
		 LIMIT 1`,
		battleID,
	))
}

func getPvpBattleByMarchTx(queryer reinforcementQueryer, marchID string) (game.PvpBattle, error) {
	return scanPvpBattle(queryer.QueryRow(
		`SELECT `+pvpBattleColumns+`
		 FROM pvp_battles
		 WHERE march_id = ?
		 LIMIT 1`,
		marchID,
	))
}

func scanPvpBattleRows(rows *sql.Rows) ([]game.PvpBattle, error) {
	items := []game.PvpBattle{}
	for rows.Next() {
		item, err := scanPvpBattle(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPvpBattle(scanner reinforcementScanner) (game.PvpBattle, error) {
	var battle game.PvpBattle
	var attackerSnapshotJSON, defenderSnapshotJSON, reinforcementSnapshotJSON, resultJSON, lossesJSON, plunderJSON sql.NullString
	var resolvedAt, createdAt, updatedAt sql.NullTime
	err := scanner.Scan(
		&battle.ID, &battle.MarchID, &battle.AttackerPlayerID, &battle.DefenderPlayerID, &battle.Status,
		&attackerSnapshotJSON, &defenderSnapshotJSON, &reinforcementSnapshotJSON,
		&resultJSON, &lossesJSON, &plunderJSON, &battle.AttackerReportID, &battle.DefenderReportID,
		&resolvedAt, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return game.PvpBattle{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.PvpBattle{}, err
	}
	unmarshalNullJSON(attackerSnapshotJSON, &battle.AttackerSnapshot)
	unmarshalNullJSON(defenderSnapshotJSON, &battle.DefenderSnapshot)
	unmarshalNullJSON(reinforcementSnapshotJSON, &battle.ReinforcementSnapshot)
	unmarshalNullJSON(resultJSON, &battle.Result)
	unmarshalNullJSON(lossesJSON, &battle.Losses)
	unmarshalNullJSON(plunderJSON, &battle.Plunder)
	battle.ResolvedAt = formatNullTime(resolvedAt)
	battle.CreatedAt = formatNullTime(createdAt)
	battle.UpdatedAt = formatNullTime(updatedAt)
	return battle, nil
}

func scanPvpPlayerState(scanner reinforcementScanner) (game.PvpPlayerState, error) {
	var state game.PvpPlayerState
	var protectedUntil, cooldownUntil, dailyResetAt, createdAt, updatedAt sql.NullTime
	var targetCooldownJSON, metadataJSON sql.NullString
	err := scanner.Scan(
		&state.PlayerID, &state.Status, &state.ProtectionType, &protectedUntil, &cooldownUntil,
		&state.DailyAttackCount, &state.DailyAttackLimit, &dailyResetAt, &targetCooldownJSON,
		&metadataJSON, &createdAt, &updatedAt,
	)
	if err != nil {
		return game.PvpPlayerState{}, err
	}
	state.ProtectedUntil = formatNullTime(protectedUntil)
	state.CooldownUntil = formatNullTime(cooldownUntil)
	state.DailyResetAt = formatNullTime(dailyResetAt)
	state.CreatedAt = formatNullTime(createdAt)
	state.UpdatedAt = formatNullTime(updatedAt)
	unmarshalNullJSON(targetCooldownJSON, &state.TargetCooldown)
	unmarshalNullJSON(metadataJSON, &state.Metadata)
	return state, nil
}

func insertBattleReportTx(tx *sql.Tx, report game.BattleReport) error {
	report = game.NormalizeBattleReport(report)
	reportJSON, err := marshalBattleReportBodyJSON(report)
	if err != nil {
		return err
	}
	detailJSON, err := json.Marshal(report.Detail)
	if err != nil {
		return err
	}
	createdAt, _ := time.Parse(time.RFC3339, report.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err = tx.Exec(
		`INSERT INTO battle_reports (
			id, player_id, event_id, owner_player_id, view_type, source_type, battle_type, result,
			title, summary, target_type, target_id, target_name, detail_json,
			report_json, type, is_read, deleted_by_player, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.PlayerID, report.EventID, report.OwnerPlayerID, report.ViewType, report.SourceType, report.BattleType, report.Result,
		report.Title, report.Summary, report.SourceType, report.TargetID, report.TargetName, detailJSON,
		reportJSON, report.Type, report.Read, false, createdAt.UTC(),
	)
	if err != nil {
		return err
	}
	if err := insertBattleEventForReportTx(tx, report, createdAt.UTC()); err != nil {
		return err
	}
	if err := insertBattleReportParticipantsForReportTx(tx, report, createdAt.UTC()); err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO battle_report_states (id, report_id, player_id, is_read, is_deleted, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 0, ?, ?)
		 ON DUPLICATE KEY UPDATE is_read = VALUES(is_read), updated_at = VALUES(updated_at)`,
		battleReportStateID(report.ID, report.PlayerID), report.ID, report.PlayerID, report.Read, createdAt.UTC(), createdAt.UTC(),
	)
	if err != nil {
		return err
	}
	return enforceBattleReportVisibleCapTx(tx, report.PlayerID, report.ViewType, time.Now().UTC())
}

// enforceBattleReportVisibleCapTx 软删除同玩家同视角超过上限的旧可见战报，保护有效分享链接。
func enforceBattleReportVisibleCapTx(tx *sql.Tx, playerID string, viewType string, now time.Time) error {
	playerID = strings.TrimSpace(playerID)
	viewType = strings.TrimSpace(viewType)
	if playerID == "" || viewType == "" {
		return nil
	}
	rows, err := tx.Query(
		`SELECT br.id
		 FROM battle_reports br
		 LEFT JOIN battle_report_links l
		   ON l.report_id = br.id AND (l.expires_at IS NULL OR l.expires_at > ?)
		 WHERE br.player_id = ? AND br.view_type = ? AND br.deleted_by_player = 0 AND l.report_id IS NULL
		 ORDER BY br.created_at DESC, br.id DESC
		 LIMIT ? OFFSET ?`,
		now.UTC(),
		playerID,
		viewType,
		battleReportCapDeleteBatchLimit,
		battleReportVisibleCapPerView,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	reportIDs := []string{}
	for rows.Next() {
		var reportID string
		if err := rows.Scan(&reportID); err != nil {
			return err
		}
		if strings.TrimSpace(reportID) != "" {
			reportIDs = append(reportIDs, reportID)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(reportIDs) == 0 {
		return nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(reportIDs)), ",")
	if _, err := tx.Exec(
		`UPDATE battle_reports
		 SET deleted_by_player = 1
		 WHERE player_id = ? AND view_type = ? AND id IN (`+placeholders+`)`,
		append([]any{playerID, viewType}, reportIDsToAny(reportIDs)...)...,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE battle_report_states
		 SET is_deleted = 1, deleted_at = ?, updated_at = ?
		 WHERE player_id = ? AND report_id IN (`+placeholders+`)`,
		append([]any{now.UTC(), now.UTC(), playerID}, reportIDsToAny(reportIDs)...)...,
	); err != nil {
		return err
	}
	return nil
}

// reportIDsToAny 把 report_id 列表转换为 SQL 参数。
func reportIDsToAny(reportIDs []string) []any {
	args := make([]any, 0, len(reportIDs))
	for _, reportID := range reportIDs {
		args = append(args, reportID)
	}
	return args
}

func savePvpPlayerJSONTx(tx *sql.Tx, playerID string, state game.GameState, previousJSON []byte, updatedAt time.Time) error {
	nextJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return err
	}
	if bytes.Equal(previousJSON, nextJSON) {
		return nil
	}
	_, err = tx.Exec(
		`UPDATE players
		 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
		 WHERE id = ?`,
		state.Player.Nickname, state.Player.Faction, state.Player.MailCode, nextJSON, updatedAt.UTC(), playerID,
	)
	return err
}
