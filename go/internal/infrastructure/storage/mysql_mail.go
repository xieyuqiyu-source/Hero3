// 本文件归口 MySQL 信函仓储方法。
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

func (r *MySQLRepository) SaveMail(mail game.Mail) error {
	attachmentsJSON, err := json.Marshal(mail.Attachments)
	if err != nil {
		return err
	}

	createdAt, _ := time.Parse(time.RFC3339, mail.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	var expiresAt any
	if mail.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, mail.ExpiresAt); err == nil {
			expiresAt = parsed.UTC()
		}
	}

	_, err = r.db.Exec(
		`INSERT INTO mails (
			id, player_id, mail_type, sender_type, sender_id, sender_name,
			title, content, attachments_json, source_type, source_id,
			is_read, is_claimed, deleted_by_player, expires_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mail.ID,
		mail.PlayerID,
		mail.MailType,
		mail.SenderType,
		mail.SenderID,
		mail.SenderName,
		mail.Title,
		mail.Content,
		attachmentsJSON,
		mail.SourceType,
		mail.SourceID,
		mail.IsRead,
		mail.IsClaimed,
		false,
		expiresAt,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) GetMailByID(mailID string) (game.Mail, error) {
	row := r.db.QueryRow(
		`SELECT id, player_id, mail_type, sender_type, sender_id, sender_name,
			title, content, attachments_json, source_type, source_id,
			is_read, is_claimed, deleted_by_player, expires_at, created_at, read_at, claimed_at
		 FROM mails WHERE id = ? LIMIT 1`,
		mailID,
	)
	return scanMail(row)
}

func (r *MySQLRepository) ListMails(playerID string, mailType string, limit int, offset int) ([]game.Mail, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	mailType = strings.TrimSpace(mailType)

	var total int
	var rows *sql.Rows
	var err error
	if mailType == "" {
		if err := r.db.QueryRow(
			`SELECT COUNT(*) FROM mails WHERE player_id = ? AND deleted_by_player = 0`,
			playerID,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.db.Query(
			`SELECT id, player_id, mail_type, sender_type, sender_id, sender_name,
				title, content, attachments_json, source_type, source_id,
				is_read, is_claimed, deleted_by_player, expires_at, created_at, read_at, claimed_at
			 FROM mails
			 WHERE player_id = ? AND deleted_by_player = 0
			 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			playerID, limit, offset,
		)
	} else {
		if err := r.db.QueryRow(
			`SELECT COUNT(*) FROM mails WHERE player_id = ? AND mail_type = ? AND deleted_by_player = 0`,
			playerID, mailType,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.db.Query(
			`SELECT id, player_id, mail_type, sender_type, sender_id, sender_name,
				title, content, attachments_json, source_type, source_id,
				is_read, is_claimed, deleted_by_player, expires_at, created_at, read_at, claimed_at
			 FROM mails
			 WHERE player_id = ? AND mail_type = ? AND deleted_by_player = 0
			 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			playerID, mailType, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	mails := []game.Mail{}
	for rows.Next() {
		mail, err := scanMail(rows)
		if err != nil {
			return nil, 0, err
		}
		mails = append(mails, mail)
	}
	return mails, total, rows.Err()
}

func (r *MySQLRepository) CountUnreadMails(playerID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM mails WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		playerID,
	).Scan(&count)
	return count, err
}

func (r *MySQLRepository) MarkMailRead(playerID string, mailID string, readAt time.Time) error {
	result, err := r.db.Exec(
		`UPDATE mails
		 SET is_read = 1, read_at = COALESCE(read_at, ?)
		 WHERE id = ? AND player_id = ? AND deleted_by_player = 0`,
		readAt.UTC(), mailID, playerID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return errors.New("mail not found")
	}
	return err
}

func (r *MySQLRepository) DeleteMail(playerID string, mailID string) error {
	result, err := r.db.Exec(
		`UPDATE mails SET deleted_by_player = 1 WHERE id = ? AND player_id = ?`,
		mailID, playerID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return errors.New("mail not found")
	}
	return err
}

func (r *MySQLRepository) UpdateMailPlayerState(playerID string, mailID string, updatedAt time.Time, update func(account *game.Account, state *game.GameState, mail *game.Mail) error) (game.Account, game.GameState, game.Mail, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	defer func() { _ = tx.Rollback() }()

	mail, err := scanMail(tx.QueryRow(
		`SELECT id, player_id, mail_type, sender_type, sender_id, sender_name,
			title, content, attachments_json, source_type, source_id,
			is_read, is_claimed, deleted_by_player, expires_at, created_at, read_at, claimed_at
		 FROM mails
		 WHERE id = ? AND player_id = ? AND deleted_by_player = 0
		 LIMIT 1 FOR UPDATE`,
		mailID, playerID,
	))
	if err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, game.ErrMailNotFound
	}

	var accountID string
	var mailCode string
	var stateJSON []byte
	err = tx.QueryRow(
		`SELECT account_id, mail_code, state_json FROM players WHERE id = ? LIMIT 1 FOR UPDATE`,
		playerID,
	).Scan(&accountID, &mailCode, &stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.Mail{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}

	var state game.GameState
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeBuildingsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeResourceSlotsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeRecruitQueuesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	previousInventorySnapshot := inventorySnapshotsFromStorageState(state.Inventory)
	previousBuildingSnapshot := buildingSnapshotsFromStorageState(state.Buildings)
	previousResourceSlotSnapshot := resourceSlotSnapshotsFromStorageState(state.ResourceSlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousRecruitQueueSnapshot := recruitQueueSnapshotsFromStorageState(state.RecruitQueues)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)

	var account game.Account
	err = tx.QueryRow(
		`SELECT id, username, password_hash, gold, created_at
		 FROM accounts
		 WHERE id = ?
		 LIMIT 1
		 FOR UPDATE`,
		accountID,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.Gold, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.Mail{}, game.ErrAccountNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	previousAccountGold := account.Gold

	if update != nil {
		if err = update(&account, &state, &mail); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, updatedAt)
	}
	game.EnsureGeneralRoster(&state, updatedAt)

	if account.Gold != previousAccountGold {
		if _, err = tx.Exec(`UPDATE accounts SET gold = ? WHERE id = ?`, account.Gold, account.ID); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}

	nextStateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	if resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if inventorySnapshotChanged(previousInventorySnapshot, state.Inventory) {
		if err := syncPlayerInventoryTx(tx, playerID, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerBuildingsTx(tx, playerID, state.Buildings, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if resourceSlotSnapshotChanged(previousResourceSlotSnapshot, state.ResourceSlots) || buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerResourceSlotsTx(tx, playerID, state.ResourceSlots, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if recruitQueueSnapshotChanged(previousRecruitQueueSnapshot, state.RecruitQueues) {
		if err := syncPlayerRecruitQueuesTx(tx, playerID, state.RecruitQueues, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if buffSnapshotChanged(previousBuffSnapshot, state.Buffs) {
		if err := syncPlayerBuffsTx(tx, playerID, state.Buffs, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		if _, err = tx.Exec(
			`UPDATE players SET state_json = ?, mail_code = ?, updated_at = ? WHERE id = ?`,
			nextStateJSON, state.Player.MailCode, updatedAt.UTC(), playerID,
		); err != nil {
			return game.Account{}, game.GameState{}, game.Mail{}, err
		}
	}

	if _, err = tx.Exec(
		`UPDATE mails
		 SET is_claimed = 1, claimed_at = ?
		 WHERE id = ? AND player_id = ? AND deleted_by_player = 0`,
		updatedAt.UTC(), mailID, playerID,
	); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}

	if err = tx.Commit(); err != nil {
		return game.Account{}, game.GameState{}, game.Mail{}, err
	}
	return account, state, mail, nil
}

type mailScanner interface {
	Scan(dest ...any) error
}

func scanMail(scanner mailScanner) (game.Mail, error) {
	var mail game.Mail
	var attachmentsJSON []byte
	var expiresAt sql.NullTime
	var createdAt time.Time
	var readAt sql.NullTime
	var claimedAt sql.NullTime

	err := scanner.Scan(
		&mail.ID,
		&mail.PlayerID,
		&mail.MailType,
		&mail.SenderType,
		&mail.SenderID,
		&mail.SenderName,
		&mail.Title,
		&mail.Content,
		&attachmentsJSON,
		&mail.SourceType,
		&mail.SourceID,
		&mail.IsRead,
		&mail.IsClaimed,
		&mail.DeletedByPlayer,
		&expiresAt,
		&createdAt,
		&readAt,
		&claimedAt,
	)
	if err != nil {
		return game.Mail{}, err
	}

	if len(attachmentsJSON) > 0 {
		_ = json.Unmarshal(attachmentsJSON, &mail.Attachments)
	}
	if expiresAt.Valid {
		mail.ExpiresAt = expiresAt.Time.UTC().Format(time.RFC3339)
	}
	mail.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if readAt.Valid {
		mail.ReadAt = readAt.Time.UTC().Format(time.RFC3339)
	}
	if claimedAt.Valid {
		mail.ClaimedAt = claimedAt.Time.UTC().Format(time.RFC3339)
	}

	return mail, nil
}

// --- MiniGame Record Methods ---
