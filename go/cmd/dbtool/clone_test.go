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
