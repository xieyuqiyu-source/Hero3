package game

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"sync"
)

// 本文件归口物品掉落池配置加载、查询和校验。

type DropPoolsConfig map[string]DropPoolDefinition

type DropPoolDefinition struct {
	ID    string           `json:"id,omitempty"`
	Rolls int              `json:"rolls"`
	Items []DropPoolReward `json:"items"`
}

type DropPoolReward struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Amount     int    `json:"amount"`
	Weight     int    `json:"weight"`
	DropPoolID string `json:"dropPoolId,omitempty"`
}

var (
	dropPoolsMu     sync.RWMutex
	dropPoolsConfig = DropPoolsConfig{}
)

// LoadDropPoolsConfig 加载掉落池配置。
func LoadDropPoolsConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg DropPoolsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if err := ValidateDropPoolsConfig(cfg); err != nil {
		return err
	}
	dropPoolsMu.Lock()
	dropPoolsConfig = cloneDropPoolsConfig(cfg)
	dropPoolsMu.Unlock()
	return nil
}

// GetDropPoolsConfig 获取当前掉落池配置。
func GetDropPoolsConfig() DropPoolsConfig {
	dropPoolsMu.RLock()
	defer dropPoolsMu.RUnlock()
	return cloneDropPoolsConfig(dropPoolsConfig)
}

// GetDropPoolDefinition 获取指定掉落池配置。
func GetDropPoolDefinition(poolID string) (DropPoolDefinition, bool) {
	poolID = strings.TrimSpace(poolID)
	dropPoolsMu.RLock()
	defer dropPoolsMu.RUnlock()
	pool, ok := dropPoolsConfig[poolID]
	if !ok {
		return DropPoolDefinition{}, false
	}
	return cloneDropPoolDefinition(pool), true
}

// ValidateDropPoolsConfig 校验掉落池配置。
func ValidateDropPoolsConfig(cfg DropPoolsConfig) error {
	if cfg == nil {
		return errors.New("掉落池配置不能为空")
	}
	for id, pool := range cfg {
		id = strings.TrimSpace(id)
		if id == "" {
			return errors.New("掉落池 ID 不能为空")
		}
		if pool.ID != "" && pool.ID != id {
			return errors.New("掉落池 ID 和配置 key 不一致: " + id)
		}
		if pool.Rolls <= 0 {
			return errors.New("掉落池抽取次数必须大于 0: " + id)
		}
		if len(pool.Items) == 0 {
			return errors.New("掉落池奖励项不能为空: " + id)
		}
		totalWeight := 0
		for _, item := range pool.Items {
			if item.Weight <= 0 {
				return errors.New("掉落项权重必须大于 0: " + id)
			}
			if item.Amount <= 0 {
				return errors.New("掉落项数量必须大于 0: " + id)
			}
			switch strings.TrimSpace(item.Type) {
			case RewardTypeItem:
				if _, ok := GetItemDefinition(item.ID); !ok {
					return errors.New("掉落项引用的物品不存在: " + id)
				}
			case "drop_pool":
				if strings.TrimSpace(item.DropPoolID) == "" || item.DropPoolID == id {
					return errors.New("掉落池不能引用自身或空掉落池: " + id)
				}
				if _, ok := cfg[item.DropPoolID]; !ok {
					return errors.New("掉落项引用的掉落池不存在: " + id)
				}
			default:
				return errors.New("掉落项类型不合法: " + id)
			}
			totalWeight += item.Weight
		}
		if totalWeight <= 0 {
			return errors.New("掉落池总权重必须大于 0: " + id)
		}
	}
	for id := range cfg {
		if hasDropPoolCycle(cfg, id, map[string]bool{}, map[string]bool{}) {
			return errors.New("掉落池存在循环引用: " + id)
		}
	}
	return nil
}

func hasDropPoolCycle(cfg DropPoolsConfig, id string, visiting map[string]bool, visited map[string]bool) bool {
	if visiting[id] {
		return true
	}
	if visited[id] {
		return false
	}
	visiting[id] = true
	pool := cfg[id]
	for _, item := range pool.Items {
		if strings.TrimSpace(item.Type) != "drop_pool" {
			continue
		}
		nextID := strings.TrimSpace(item.DropPoolID)
		if nextID != "" && hasDropPoolCycle(cfg, nextID, visiting, visited) {
			return true
		}
	}
	visiting[id] = false
	visited[id] = true
	return false
}

func cloneDropPoolsConfig(source DropPoolsConfig) DropPoolsConfig {
	next := make(DropPoolsConfig, len(source))
	for id, pool := range source {
		next[id] = cloneDropPoolDefinition(pool)
	}
	return next
}

func cloneDropPoolDefinition(source DropPoolDefinition) DropPoolDefinition {
	next := source
	next.Items = append([]DropPoolReward(nil), source.Items...)
	return next
}

// RollDropPoolRewards 按权重抽取掉落池并转换为标准奖励。
func RollDropPoolRewards(poolID string) ([]Reward, error) {
	pool, ok := GetDropPoolDefinition(poolID)
	if !ok {
		return nil, ErrDropPoolNotFound
	}
	rewards := []Reward{}
	for i := 0; i < pool.Rolls; i++ {
		item, err := pickDropPoolReward(pool)
		if err != nil {
			return nil, err
		}
		switch strings.TrimSpace(item.Type) {
		case RewardTypeItem:
			rewards = append(rewards, Reward{Type: RewardTypeItem, ID: strings.TrimSpace(item.ID), Amount: item.Amount})
		case "drop_pool":
			nested, err := RollDropPoolRewards(item.DropPoolID)
			if err != nil {
				return nil, err
			}
			rewards = append(rewards, nested...)
		default:
			return nil, ErrDropPoolNotFound
		}
	}
	return rewards, nil
}

func pickDropPoolReward(pool DropPoolDefinition) (DropPoolReward, error) {
	total := 0
	for _, item := range pool.Items {
		total += item.Weight
	}
	if total <= 0 {
		return DropPoolReward{}, ErrDropPoolNotFound
	}
	roll, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return DropPoolReward{}, err
	}
	cursor := int(roll.Int64()) + 1
	for _, item := range pool.Items {
		cursor -= item.Weight
		if cursor <= 0 {
			return item, nil
		}
	}
	return pool.Items[len(pool.Items)-1], nil
}
