// 本文件测试资产级事务不再在提交后回读完整玩家状态。
package storage

import (
	"os"
	"strings"
	"testing"
)

// TestAssetTransactionsDoNotCallFullGetStateAfterCommit 防止高频资产事务重新引入完整状态组装。
func TestAssetTransactionsDoNotCallFullGetStateAfterCommit(t *testing.T) {
	files := []string{
		"mysql_building_assets.go",
		"mysql_item_assets.go",
		"mysql_recruit_assets.go",
		"mysql_general_assets.go",
		"mysql_reward_assets.go",
		"mysql_minigame.go",
		"mysql_reinforcements.go",
		"mysql_player_meta.go",
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(content), "r.GetState(") {
			t.Fatalf("%s must not call r.GetState in asset-level transactions", file)
		}
	}
}
