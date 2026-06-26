// 本文件验证 MySQL 小游戏记录仓储的写入判断辅助逻辑。
package storage

import (
	"testing"

	"hero3/internal/app/game"
)

func TestMiniGameRemainingAmountChanged(t *testing.T) {
	previous := map[string]int{
		"record_same":    100,
		"record_changed": 100,
	}

	if miniGameRemainingAmountChanged(previous, game.MiniGameRecord{ID: "record_same", RemainingAmount: 100}) {
		t.Fatal("expected unchanged remaining amount to be skipped")
	}
	if !miniGameRemainingAmountChanged(previous, game.MiniGameRecord{ID: "record_changed", RemainingAmount: 80}) {
		t.Fatal("expected changed remaining amount to be updated")
	}
	if !miniGameRemainingAmountChanged(previous, game.MiniGameRecord{ID: "record_new", RemainingAmount: 50}) {
		t.Fatal("expected unknown record to keep update-compatible behavior")
	}
}
