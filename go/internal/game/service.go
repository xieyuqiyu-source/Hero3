package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrAccountExists      = errors.New("account already exists")
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPlayerNotFound     = errors.New("player not found")
)

type Service struct {
	repo Repository
}

type BootstrapResponse struct {
	GameName string   `json:"gameName"`
	Modules  []string `json:"modules"`
	Message  string   `json:"message"`
}

func NewService() *Service {
	return NewServiceWithRepository(NewMemoryRepository())
}

func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterAccount(username string, password string) (Account, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return Account{}, ErrInvalidCredentials
	}

	account := Account{
		ID:           "acc_" + randomID(12),
		Username:     username,
		PasswordHash: hashPassword(password),
		CreatedAt:    time.Now(),
	}
	if err := s.repo.CreateAccount(account); err != nil {
		return Account{}, err
	}

	return account, nil
}

func (s *Service) LoginAccount(username string, password string) (Account, error) {
	username = strings.TrimSpace(username)

	account, err := s.repo.GetAccountByUsername(username)
	if err != nil {
		return Account{}, ErrInvalidCredentials
	}

	if account.PasswordHash != hashPassword(password) {
		return Account{}, ErrInvalidCredentials
	}

	return account, nil
}

func (s *Service) ListPlayers(accountID string) ([]PlayerSummary, error) {
	return s.repo.ListPlayers(accountID)
}

func (s *Service) ListAccounts() ([]AccountSummary, error) {
	return s.repo.ListAccounts()
}

func (s *Service) CreatePlayer(accountID string, nickname string, faction string) (string, GameState, error) {
	nickname = strings.TrimSpace(nickname)
	faction = strings.TrimSpace(faction)
	if nickname == "" || faction == "" {
		return "", GameState{}, ErrPlayerNotFound
	}

	exists, err := s.repo.AccountExists(accountID)
	if err != nil {
		return "", GameState{}, err
	}
	if !exists {
		return "", GameState{}, ErrAccountNotFound
	}

	now := time.Now()
	playerID := "player_" + randomID(12)
	state := newPlayerState(playerID, nickname, faction, now)
	if err := s.repo.CreatePlayer(accountID, state, now); err != nil {
		return "", GameState{}, err
	}

	return playerID, state, nil
}

func (s *Service) GetState(playerID string) (GameState, error) {
	if strings.TrimSpace(playerID) == "" {
		playerID = "demo-player"
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) && playerID == "demo-player" {
			state = newDemoState(time.Now())
			return state, nil
		}
		return GameState{}, err
	}

	state.ServerTime = time.Now().UTC().Format(time.RFC3339)
	return state, nil
}

func (s *Service) Bootstrap() BootstrapResponse {
	return BootstrapResponse{
		GameName: "Hero3",
		Modules: []string{
			"player",
			"city",
			"resource",
			"military",
			"map",
			"combat",
			"save",
		},
		Message: "Hero3 后端基础服务已就绪，具体玩法逻辑待接入。",
	}
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func randomID(bytesCount int) string {
	bytes := make([]byte, bytesCount)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
