package game

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type SendMailRequest struct {
	PlayerID    string           `json:"playerId"`
	MailType    string           `json:"mailType"`
	SenderType  string           `json:"senderType"`
	SenderID    string           `json:"senderId"`
	SenderName  string           `json:"senderName"`
	Title       string           `json:"title"`
	Content     string           `json:"content"`
	Attachments []MailAttachment `json:"attachments"`
	SourceType  string           `json:"sourceType"`
	SourceID    string           `json:"sourceId"`
	ExpiresAt   string           `json:"expiresAt"`
}

type SendPlayerMailRequest struct {
	SenderPlayerID string `json:"senderPlayerId"`
	Recipient      string `json:"recipient"`
	Title          string `json:"title"`
	Content        string `json:"content"`
}

type SendServerBroadcastMailRequest struct {
	SenderPlayerID string `json:"senderPlayerId"`
	Title          string `json:"title"`
	Content        string `json:"content"`
}

type SendServerBroadcastMailResult struct {
	Cost           int     `json:"cost"`
	CityGold       FlexInt `json:"cityGold"`
	RecipientCount int     `json:"recipientCount"`
	ServerTime     string  `json:"serverTime"`
}

const (
	ServerBroadcastMailType = "server_broadcast"
	ServerBroadcastCost     = 100000
)

var mailAddressPattern = regexp.MustCompile(`^(.+)#([0-9]{6})$`)

func (s *Service) ListMails(playerID string, page int, pageSize int, mailType string) (MailPage, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return MailPage{}, ErrPlayerNotFound
	}
	mailType, ok := normalizeMailTypeFilter(mailType)
	if !ok {
		return MailPage{}, ErrInvalidMail
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	mails, total, err := s.repo.ListMails(playerID, mailType, pageSize, offset)
	if err != nil {
		return MailPage{}, err
	}
	unread, err := s.repo.CountUnreadMails(playerID)
	if err != nil {
		return MailPage{}, err
	}
	return MailPage{Mails: mails, Page: page, PageSize: pageSize, Total: total, Unread: unread}, nil
}

func (s *Service) GetMail(playerID string, mailID string) (Mail, error) {
	playerID = strings.TrimSpace(playerID)
	mailID = strings.TrimSpace(mailID)
	if playerID == "" {
		return Mail{}, ErrPlayerNotFound
	}
	if mailID == "" {
		return Mail{}, ErrMailNotFound
	}

	mail, err := s.repo.GetMailByID(mailID)
	if err != nil {
		return Mail{}, ErrMailNotFound
	}
	if mail.PlayerID != playerID || mail.DeletedByPlayer {
		return Mail{}, ErrMailNotFound
	}
	if !mail.IsRead {
		now := time.Now()
		if err := s.repo.MarkMailRead(playerID, mailID, now); err != nil {
			return Mail{}, err
		}
		mail.IsRead = true
		mail.ReadAt = now.UTC().Format(resourceDateLayout)
	}
	return mail, nil
}

func (s *Service) DeleteMail(playerID string, mailID string) error {
	playerID = strings.TrimSpace(playerID)
	mailID = strings.TrimSpace(mailID)
	if playerID == "" {
		return ErrPlayerNotFound
	}
	if mailID == "" {
		return ErrMailNotFound
	}
	return s.repo.DeleteMail(playerID, mailID)
}

func (s *Service) ClaimMailAttachments(playerID string, mailID string) (MailClaimResult, error) {
	playerID = strings.TrimSpace(playerID)
	mailID = strings.TrimSpace(mailID)
	if playerID == "" {
		return MailClaimResult{}, ErrPlayerNotFound
	}
	if mailID == "" {
		return MailClaimResult{}, ErrMailNotFound
	}

	claimedAt := time.Now()
	ctx := RewardGrantContext{
		PlayerID: playerID,
		RefType:  LedgerRefMailClaim,
		RefID:    mailID,
		Reason:   "mail_claim",
	}
	var applyResult RewardApplyResult
	account, state, mail, err := s.repo.UpdateMailPlayerState(playerID, mailID, claimedAt, func(account *Account, state *GameState, mail *Mail) error {
		if mail.PlayerID != playerID || mail.DeletedByPlayer {
			return ErrMailNotFound
		}
		if len(mail.Attachments) == 0 {
			return ErrMailNoAttachments
		}
		if mail.IsClaimed {
			return ErrMailAlreadyClaimed
		}
		if mail.SenderType == "player" || mail.MailType == "player_message" {
			return ErrMailClaimForbidden
		}
		ctx.AccountID = account.ID
		effectResult, err := ExecuteEffectsOnState(state, rewardsToEffects("mail", rewardsFromMailAttachments(mail.Attachments)), EffectContext{
			AccountID: ctx.AccountID,
			PlayerID:  ctx.PlayerID,
			RefType:   ctx.RefType,
			RefID:     ctx.RefID,
			Reason:    ctx.Reason,
			Source:    "mail",
		}, claimedAt)
		if err != nil {
			return err
		}
		result := effectResult.Reward
		if result.AccountGold > 0 {
			account.Gold += result.AccountGold
			result.LedgerEntries = append(result.LedgerEntries, GoldLedgerEntry{
				AccountID:    account.ID,
				PlayerID:     playerID,
				Currency:     LedgerCurrencyGold,
				Direction:    LedgerDirectionCredit,
				Amount:       result.AccountGold,
				BalanceAfter: account.Gold,
				RefType:      ctx.RefType,
				RefID:        ctx.RefID,
				Reason:       ctx.Reason,
				CreatedAt:    claimedAt.UTC().Format(resourceDateLayout),
			})
		}
		applyResult = result
		mail.IsClaimed = true
		mail.ClaimedAt = claimedAt.UTC().Format(resourceDateLayout)
		state.ServerTime = claimedAt.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return MailClaimResult{}, err
	}
	s.flushRewardSideEffects(applyResult)
	result := MailClaimResult{
		Mail:         mail,
		Resources:    state.Resources,
		Inventory:    state.Inventory,
		CityGold:     int(state.CityGold),
		AccountGold:  account.Gold,
		GrantedItems: applyResult.Granted,
	}
	s.publishMailClaimEvents(result)
	return result, nil
}

func (s *Service) SendMail(req SendMailRequest) (Mail, error) {
	now := time.Now()
	mail, err := buildMail(req, now)
	if err != nil {
		return Mail{}, err
	}
	if _, err := s.repo.GetState(mail.PlayerID); err != nil {
		return Mail{}, err
	}
	if err := s.repo.SaveMail(mail); err != nil {
		return Mail{}, err
	}
	return mail, nil
}

func (s *Service) SendPlayerMail(req SendPlayerMailRequest) (Mail, error) {
	senderID := strings.TrimSpace(req.SenderPlayerID)
	if senderID == "" {
		return Mail{}, ErrPlayerNotFound
	}
	now := time.Now()
	senderState, err := s.repo.GetState(senderID)
	if err != nil {
		return Mail{}, err
	}
	if senderState.Player.MailCode == "" {
		code, err := s.generateMailCode(senderState.Player.Nickname)
		if err != nil {
			return Mail{}, err
		}
		senderState, err = s.repo.UpdatePlayerMetaState(senderID, now, func(state *GameState) error {
			if state.Player.MailCode == "" {
				state.Player.MailCode = code
			}
			return nil
		})
		if err != nil {
			return Mail{}, err
		}
	}

	nickname, mailCode, err := parseMailAddress(req.Recipient)
	if err != nil {
		return Mail{}, ErrInvalidMail
	}
	recipient, err := s.repo.FindPlayerByMailAddress(nickname, mailCode)
	if err != nil {
		return Mail{}, ErrPlayerNotFound
	}
	if recipient.ID == senderID {
		return Mail{}, ErrMailRecipientSelf
	}

	senderName := formatMailAddress(senderState.Player.Nickname, senderState.Player.MailCode)
	return s.SendMail(SendMailRequest{
		PlayerID:    recipient.ID,
		MailType:    "player_message",
		SenderType:  "player",
		SenderID:    senderID,
		SenderName:  senderName,
		Title:       req.Title,
		Content:     req.Content,
		Attachments: nil,
		SourceType:  "player",
		SourceID:    senderID,
	})
}

// SendServerBroadcastMail 消耗城金后向全服玩家投递一封玩家喊话信函。
func (s *Service) SendServerBroadcastMail(req SendServerBroadcastMailRequest) (SendServerBroadcastMailResult, error) {
	senderID := strings.TrimSpace(req.SenderPlayerID)
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if senderID == "" {
		return SendServerBroadcastMailResult{}, ErrPlayerNotFound
	}
	if title == "" || content == "" || utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(content) > 5000 {
		return SendServerBroadcastMailResult{}, ErrInvalidMail
	}
	recipients, err := s.repo.ListAllPlayers()
	if err != nil {
		return SendServerBroadcastMailResult{}, err
	}
	if len(recipients) == 0 {
		return SendServerBroadcastMailResult{}, ErrPlayerNotFound
	}

	now := time.Now()
	var senderState GameState
	state, err := s.repo.UpdateRewardState(senderID, now, func(state *GameState) error {
		if int(state.CityGold) < ServerBroadcastCost {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(ServerBroadcastCost)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		senderState = *state
		return nil
	})
	if err != nil {
		return SendServerBroadcastMailResult{}, err
	}
	if senderState.Player.ID == "" {
		senderState = state
	}

	broadcastID := "server_broadcast_" + randomID(12)
	senderName := formatMailAddress(senderState.Player.Nickname, senderState.Player.MailCode)
	for _, recipient := range recipients {
		if strings.TrimSpace(recipient.ID) == "" {
			continue
		}
		if _, err := s.SendMail(SendMailRequest{
			PlayerID:    recipient.ID,
			MailType:    ServerBroadcastMailType,
			SenderType:  "player",
			SenderID:    senderID,
			SenderName:  "全服喊话 · " + senderName,
			Title:       title,
			Content:     content,
			SourceType:  "player",
			SourceID:    broadcastID,
			Attachments: nil,
		}); err != nil {
			return SendServerBroadcastMailResult{}, err
		}
	}

	s.recordLedger(GoldLedgerEntry{
		PlayerID:     senderID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       ServerBroadcastCost,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefServerBroadcast,
		RefID:        broadcastID,
		Reason:       "server_broadcast_mail",
		CreatedAt:    now.UTC().Format(resourceDateLayout),
	})
	s.publishCurrencyChanged(senderID, "", broadcastID, LedgerRefServerBroadcast)

	return SendServerBroadcastMailResult{
		Cost:           ServerBroadcastCost,
		CityGold:       state.CityGold,
		RecipientCount: len(recipients),
		ServerTime:     state.ServerTime,
	}, nil
}

func buildMail(req SendMailRequest, now time.Time) (Mail, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	mailType := normalizeMailType(req.MailType)
	senderType := normalizeSenderType(req.SenderType)
	senderName := strings.TrimSpace(req.SenderName)
	sourceType := normalizeSourceType(req.SourceType)

	if playerID == "" || title == "" || content == "" {
		return Mail{}, ErrInvalidMail
	}
	if utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(content) > 5000 {
		return Mail{}, ErrInvalidMail
	}
	if !validateMailAttachments(req.Attachments) {
		return Mail{}, ErrInvalidMail
	}
	if senderName == "" {
		if senderType == "gm" {
			senderName = "Hero3 GM"
		} else {
			senderName = "系统"
		}
	}

	expiresAt := ""
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			return Mail{}, ErrInvalidMail
		}
		expiresAt = parsed.UTC().Format(resourceDateLayout)
	}

	return Mail{
		ID:          "mail_" + randomID(12),
		PlayerID:    playerID,
		MailType:    mailType,
		SenderType:  senderType,
		SenderID:    strings.TrimSpace(req.SenderID),
		SenderName:  senderName,
		Title:       title,
		Content:     content,
		Attachments: req.Attachments,
		SourceType:  sourceType,
		SourceID:    strings.TrimSpace(req.SourceID),
		IsRead:      false,
		IsClaimed:   len(req.Attachments) == 0,
		ExpiresAt:   expiresAt,
		CreatedAt:   now.UTC().Format(resourceDateLayout),
	}, nil
}

func normalizeMailType(mailType string) string {
	switch strings.TrimSpace(mailType) {
	case "compensation", "reward", "event_reward", "system_notice", "player_message", PvpSeasonRewardMailType, ServerBroadcastMailType:
		return strings.TrimSpace(mailType)
	default:
		return "gm_notice"
	}
}

func normalizeMailTypeFilter(mailType string) (string, bool) {
	switch strings.TrimSpace(mailType) {
	case "", "all":
		return "", true
	case "gm_notice", "compensation", "reward", "event_reward", "system_notice", "player_message", PvpSeasonRewardMailType, ServerBroadcastMailType:
		return strings.TrimSpace(mailType), true
	default:
		return "", false
	}
}

func normalizeSenderType(senderType string) string {
	switch strings.TrimSpace(senderType) {
	case "system", "player":
		return strings.TrimSpace(senderType)
	default:
		return "gm"
	}
}

func normalizeSourceType(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case "manual", "compensation", "activity", "system", "player":
		return strings.TrimSpace(sourceType)
	default:
		return "manual"
	}
}

func validateMailAttachments(attachments []MailAttachment) bool {
	for _, attachment := range attachments {
		if attachment.Amount <= 0 {
			return false
		}
		switch strings.TrimSpace(attachment.Type) {
		case "resource":
			switch strings.TrimSpace(attachment.ItemID) {
			case "wood", "stone", "iron", "food":
			default:
				return false
			}
		case "city_gold":
			if strings.TrimSpace(attachment.ItemID) != "city_gold" {
				return false
			}
		case "gold":
			if strings.TrimSpace(attachment.ItemID) != "gold" {
				return false
			}
		case "item":
			if _, ok := GetItemDefinition(strings.TrimSpace(attachment.ItemID)); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func parseMailAddress(address string) (string, string, error) {
	address = strings.TrimSpace(address)
	matches := mailAddressPattern.FindStringSubmatch(address)
	if len(matches) != 3 {
		return "", "", ErrInvalidMail
	}
	nickname := strings.TrimSpace(matches[1])
	mailCode := strings.TrimSpace(matches[2])
	if nickname == "" || mailCode == "" {
		return "", "", ErrInvalidMail
	}
	return nickname, mailCode, nil
}

func formatMailAddress(nickname string, mailCode string) string {
	return fmt.Sprintf("%s#%s", strings.TrimSpace(nickname), strings.TrimSpace(mailCode))
}

func (s *Service) publishMailClaimEvents(result MailClaimResult) {
	claimedAt := firstNonEmpty(result.Mail.ClaimedAt, time.Now().UTC().Format(resourceDateLayout))
	s.publishEvent(GameEvent{
		Type:     EventMailClaimed,
		PlayerID: result.Mail.PlayerID,
		RefType:  LedgerRefMailClaim,
		RefID:    result.Mail.ID,
		Payload: map[string]any{
			"mailType":      result.Mail.MailType,
			"sourceType":    result.Mail.SourceType,
			"sourceId":      result.Mail.SourceID,
			"grantedItems":  result.GrantedItems,
			"cityGold":      result.CityGold,
			"accountGold":   result.AccountGold,
			"attachmentNum": len(result.Mail.Attachments),
		},
		CreatedAt: claimedAt,
	})
}
