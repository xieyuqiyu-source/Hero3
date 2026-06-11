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

	mail, err := s.repo.GetMailByID(mailID)
	if err != nil || mail.PlayerID != playerID || mail.DeletedByPlayer {
		return MailClaimResult{}, ErrMailNotFound
	}
	if len(mail.Attachments) == 0 {
		return MailClaimResult{}, ErrMailNoAttachments
	}
	if mail.IsClaimed {
		return MailClaimResult{}, ErrMailAlreadyClaimed
	}
	if mail.SenderType == "player" || mail.MailType == "player_message" {
		return MailClaimResult{}, ErrInvalidMail
	}
	if strings.TrimSpace(mail.ExpiresAt) != "" {
		expiresAt, parseErr := time.Parse(resourceDateLayout, mail.ExpiresAt)
		if parseErr == nil && time.Now().After(expiresAt) {
			return MailClaimResult{}, ErrInvalidMail
		}
	}

	result, err := s.repo.ClaimMailAttachments(playerID, mailID, time.Now())
	if err != nil {
		return MailClaimResult{}, err
	}
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
	senderState, err := s.repo.GetState(senderID)
	if err != nil {
		return Mail{}, err
	}
	if senderState.Player.MailCode == "" {
		code, err := s.generateMailCode(senderState.Player.Nickname)
		if err != nil {
			return Mail{}, err
		}
		senderState.Player.MailCode = code
		if err := s.repo.SaveState(senderState, time.Now()); err != nil {
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
	case "compensation", "reward", "event_reward", "system_notice", "player_message":
		return strings.TrimSpace(mailType)
	default:
		return "gm_notice"
	}
}

func normalizeMailTypeFilter(mailType string) (string, bool) {
	switch strings.TrimSpace(mailType) {
	case "", "all":
		return "", true
	case "gm_notice", "compensation", "reward", "event_reward", "system_notice", "player_message":
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

func ApplyMailAttachmentsToState(state *GameState, attachments []MailAttachment) (map[string]int, int, error) {
	if state == nil {
		return nil, 0, ErrPlayerNotFound
	}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}
	granted := map[string]int{}
	accountGold := 0
	for _, attachment := range attachments {
		if attachment.Amount <= 0 {
			return nil, 0, ErrInvalidMail
		}
		key := strings.TrimSpace(attachment.ItemID)
		switch strings.TrimSpace(attachment.Type) {
		case "resource":
			capacity := state.Resources.Capacity[key]
			current := state.Resources.Items[key]
			next := current + attachment.Amount
			if capacity > 0 && next > capacity {
				next = capacity
			}
			state.Resources.Items[key] = next
			granted[key] += next - current
		case "city_gold":
			state.CityGold += FlexInt(attachment.Amount)
			granted["city_gold"] += attachment.Amount
		case "gold":
			accountGold += attachment.Amount
			granted["gold"] += attachment.Amount
		default:
			return nil, 0, ErrInvalidMail
		}
	}
	return granted, accountGold, nil
}
