// 本文件验证数据库复制工具的结构兼容判断。
package main

import (
	"reflect"
	"testing"
)

func TestCloneableColumnsAllowsTargetExtraColumns(t *testing.T) {
	columns, skipped, err := cloneableColumns(
		"players",
		[]string{"id", "account_id", "state_json"},
		[]string{"id", "account_id", "state_json", "new_shadow_column"},
	)
	if err != nil {
		t.Fatalf("cloneableColumns failed: %v", err)
	}
	wantColumns := []string{"id", "account_id", "state_json"}
	if !reflect.DeepEqual(columns, wantColumns) {
		t.Fatalf("expected clone columns %v, got %v", wantColumns, columns)
	}
	wantSkipped := []string{"new_shadow_column"}
	if !reflect.DeepEqual(skipped, wantSkipped) {
		t.Fatalf("expected skipped columns %v, got %v", wantSkipped, skipped)
	}
}

func TestCloneableColumnsRejectsSourceExtraColumns(t *testing.T) {
	_, _, err := cloneableColumns(
		"players",
		[]string{"id", "state_json", "legacy_only"},
		[]string{"id", "state_json"},
	)
	if err == nil {
		t.Fatal("expected source-only column to fail")
	}
}

func TestCloneableColumnsRejectsEmptySourceColumns(t *testing.T) {
	_, _, err := cloneableColumns("empty_table", nil, []string{"id"})
	if err == nil {
		t.Fatal("expected empty source columns to fail")
	}
}

func TestBuildCloneTablePlanTruncatesTargetTables(t *testing.T) {
	sourceTables := []string{"accounts", "players"}
	targetTables := []string{"accounts", "players", "player_resources"}

	plan := buildCloneTablePlan(sourceTables, targetTables)
	if !reflect.DeepEqual(plan.CopyTables, sourceTables) {
		t.Fatalf("expected copy tables %v, got %v", sourceTables, plan.CopyTables)
	}
	if !reflect.DeepEqual(plan.TruncateTables, targetTables) {
		t.Fatalf("expected truncate tables %v, got %v", targetTables, plan.TruncateTables)
	}

	sourceTables[0] = "mutated"
	targetTables[0] = "mutated"
	if plan.CopyTables[0] == "mutated" || plan.TruncateTables[0] == "mutated" {
		t.Fatal("expected clone table plan to copy table slices")
	}
}
