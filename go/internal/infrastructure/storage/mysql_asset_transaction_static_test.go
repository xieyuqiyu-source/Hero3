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

// TestPlayerInventoryDeletesStaySlotScoped 防止背包同步重新引入玩家级大范围删除锁。
func TestPlayerInventoryDeletesStaySlotScoped(t *testing.T) {
	files := []string{
		"mysql_player_state.go",
		"mysql_combat_assets.go",
		"mysql_item_assets.go",
		"mysql_general_assets.go",
		"mysql_general_exp_item_assets.go",
		"mysql_reward_assets.go",
		"mysql_mail.go",
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source := string(content)
		if file == "mysql_player_state.go" {
			source = strings.Split(source, "func deletePlayerRelatedRowsTx")[0]
		}
		normalized := strings.Join(strings.Fields(source), " ")
		if strings.Contains(normalized, "DELETE FROM player_inventory WHERE player_id = ?`,") {
			t.Fatalf("%s must not delete all player_inventory rows by player_id", file)
		}
		if strings.Contains(normalized, "DELETE FROM player_inventory WHERE player_id = ? AND slot_id NOT IN") {
			t.Fatalf("%s must not delete player_inventory rows with NOT IN", file)
		}
	}
}
