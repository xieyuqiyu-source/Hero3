package game

import (
	"sync"
	"time"
)

type Repository interface {
	CreateAccount(account Account) error
	GetAccountByUsername(username string) (Account, error)
	AccountExists(accountID string) (bool, error)
	ListAccounts() ([]AccountSummary, error)
	ListPlayers(accountID string) ([]PlayerSummary, error)
	CreatePlayer(accountID string, state GameState, updatedAt time.Time) error
	DeleteAccount(accountID string) error
	DeletePlayer(playerID string) error
	GetState(playerID string) (GameState, error)
	SaveState(state GameState, updatedAt time.Time) error
}

type MemoryRepository struct {
	mu              sync.RWMutex
	accounts        map[string]Account
	accountByName   map[string]string
	accountPlayers  map[string][]string
	players         map[string]GameState
	playerUpdatedAt map[string]time.Time
}

func NewMemoryRepository() *MemoryRepository {
	now := time.Now()
	demoState := newDemoState(now)

	return &MemoryRepository{
		accounts:        make(map[string]Account),
		accountByName:   make(map[string]string),
		accountPlayers:  make(map[string][]string),
		players:         map[string]GameState{demoState.Player.ID: demoState},
		playerUpdatedAt: map[string]time.Time{demoState.Player.ID: now},
	}
}

func (r *MemoryRepository) CreateAccount(account Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accountByName[account.Username]; exists {
		return ErrAccountExists
	}

	r.accounts[account.ID] = account
	r.accountByName[account.Username] = account.ID
	r.accountPlayers[account.ID] = []string{}
	return nil
}

func (r *MemoryRepository) GetAccountByUsername(username string) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accountID, exists := r.accountByName[username]
	if !exists {
		return Account{}, ErrAccountNotFound
	}

	return r.accounts[accountID], nil
}

func (r *MemoryRepository) AccountExists(accountID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.accounts[accountID]
	return exists, nil
}

func (r *MemoryRepository) ListAccounts() ([]AccountSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accounts := make([]AccountSummary, 0, len(r.accounts))
	for _, account := range r.accounts {
		playerIDs := r.accountPlayers[account.ID]
		players := make([]PlayerSummary, 0, len(playerIDs))
		for _, playerID := range playerIDs {
			state, exists := r.players[playerID]
			if !exists {
				continue
			}
			players = append(players, buildPlayerSummary(state, r.playerUpdatedAt[playerID]))
		}

		accounts = append(accounts, AccountSummary{
			ID:        account.ID,
			Username:  account.Username,
			CreatedAt: account.CreatedAt.UTC().Format(time.RFC3339),
			Players:   players,
		})
	}

	return accounts, nil
}

func (r *MemoryRepository) ListPlayers(accountID string) ([]PlayerSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.accounts[accountID]; !exists {
		return nil, ErrAccountNotFound
	}

	playerIDs := r.accountPlayers[accountID]
	players := make([]PlayerSummary, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		state, exists := r.players[playerID]
		if !exists {
			continue
		}
		players = append(players, buildPlayerSummary(state, r.playerUpdatedAt[playerID]))
	}

	return players, nil
}

func (r *MemoryRepository) CreatePlayer(accountID string, state GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accounts[accountID]; !exists {
		return ErrAccountNotFound
	}

	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	r.accountPlayers[accountID] = append(r.accountPlayers[accountID], state.Player.ID)
	return nil
}

func (r *MemoryRepository) DeleteAccount(accountID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}

	for _, playerID := range r.accountPlayers[accountID] {
		delete(r.players, playerID)
		delete(r.playerUpdatedAt, playerID)
	}
	delete(r.accountPlayers, accountID)
	delete(r.accountByName, account.Username)
	delete(r.accounts, accountID)
	return nil
}

func (r *MemoryRepository) DeletePlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[playerID]; !exists {
		return ErrPlayerNotFound
	}

	delete(r.players, playerID)
	delete(r.playerUpdatedAt, playerID)
	for accountID, playerIDs := range r.accountPlayers {
		nextPlayerIDs := playerIDs[:0]
		for _, currentID := range playerIDs {
			if currentID != playerID {
				nextPlayerIDs = append(nextPlayerIDs, currentID)
			}
		}
		r.accountPlayers[accountID] = nextPlayerIDs
	}
	return nil
}

func (r *MemoryRepository) GetState(playerID string) (GameState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.players[playerID]
	if !exists {
		return GameState{}, ErrPlayerNotFound
	}

	return state, nil
}

func (r *MemoryRepository) SaveState(state GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[state.Player.ID]; !exists {
		return ErrPlayerNotFound
	}

	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	return nil
}
