// 本文件实现 GM 配置从文件模板到数据库配置源的装载逻辑。
package game

import (
	"encoding/json"
	"time"
)

// loadFishingConfigFromRepository 用数据库中的线上配置覆盖文件模板；数据库缺失时种入文件模板。
func (s *Service) loadFishingConfigFromRepository(fileConfig FishingConfig) error {
	record, exists, err := s.repo.GetGameConfig(gameConfigKeyFishing)
	if err != nil {
		return err
	}
	if exists {
		var stored FishingConfig
		if err := json.Unmarshal(record.ValueJSON, &stored); err != nil {
			return err
		}
		return SetFishingConfig(stored)
	}

	content, err := json.Marshal(fileConfig)
	if err != nil {
		return err
	}
	_, err = s.repo.SaveGameConfig(gameConfigKeyFishing, content, "system_seed", time.Now().UTC())
	return err
}
