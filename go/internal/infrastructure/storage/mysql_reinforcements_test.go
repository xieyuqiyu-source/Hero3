// 本文件测试增援 MySQL 仓储的展示信息补齐逻辑。
package storage

import (
	"os"
	"strings"
	"testing"

	"hero3/internal/app/game"
)

// TestReinforcementTransactionsSyncAuthoritativeGenerals 防止增援发奖只写玩家 JSON 而遗漏权威武将表。
func TestReinforcementTransactionsSyncAuthoritativeGenerals(t *testing.T) {
	content, err := os.ReadFile("mysql_reinforcements.go")
	if err != nil {
		t.Fatalf("read mysql_reinforcements.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "generalSnapshotChanged(previousGenerals, state.Generals)") || !strings.Contains(source, "syncPlayerGeneralsTx(tx, playerID, state.Generals") {
		t.Fatal("reinforcement player transaction must sync changed generals to player_generals")
	}
}

// TestApplyReinforcementPlayerLabels 验证增援记录能补齐玩家名且不覆盖已有快照名。
func TestApplyReinforcementPlayerLabels(t *testing.T) {
	records := []game.Reinforcement{
		{FromPlayerID: "player_from", ToPlayerID: "player_to"},
		{FromPlayerID: "player_old", FromPlayerName: "旧名快照", ToPlayerID: "player_to"},
	}
	labels := map[string]reinforcementPlayerLabel{
		"player_from": {Nickname: "援军城", Faction: "shu"},
		"player_to":   {Nickname: "目标城", Faction: "wei"},
		"player_old":  {Nickname: "新名", Faction: "wu"},
	}

	applyReinforcementPlayerLabels(records, labels)

	if records[0].FromPlayerName != "援军城" || records[0].FromPlayerFaction != "shu" {
		t.Fatalf("expected from player label filled, got %+v", records[0])
	}
	if records[0].ToPlayerName != "目标城" || records[0].ToPlayerFaction != "wei" {
		t.Fatalf("expected to player label filled, got %+v", records[0])
	}
	if records[1].FromPlayerName != "旧名快照" {
		t.Fatalf("expected existing from player snapshot name kept, got %s", records[1].FromPlayerName)
	}
	if records[1].FromPlayerFaction != "wu" {
		t.Fatalf("expected missing faction filled, got %s", records[1].FromPlayerFaction)
	}
}
