package game

import (
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

func (s *Service) ListMails(playerID string, page int, pageSize int) (MailPage, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return MailPage{}, ErrPlayerNotFound
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
	mails, total, err := s.repo.ListMails(playerID, pageSize, offset)
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
