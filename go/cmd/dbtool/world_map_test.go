// 本文件验证世界地图 dbtool 维护命令的可发现性和正式库保护参数。
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackfillWorldPositionsCommandOptions 锁定世界地图坐标回填命令的正式库保护和 Makefile 入口。
func TestBackfillWorldPositionsCommandOptions(t *testing.T) {
	source, err := os.ReadFile("world_map.go")
	if err != nil {
		t.Fatalf("read world_map.go: %v", err)
	}
	content := string(source)
	for _, required := range []string{
		`flags.String("dsn"`,
		`flags.Bool("allow-non-test"`,
		"resolveWritableDbtoolDSN(*dsn, *allowNonTest)",
		"context.WithTimeout(context.Background(), longCommandTimeout)",
		"database=%s total=%d created=%d skipped=%d conflicts=%d failed=%d",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("backfill-world-positions missing %q", required)
		}
	}

	makefile, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileContent := string(makefile)
	for _, required := range []string{
		"backfill-world-positions",
		"cd go && go run ./cmd/dbtool backfill-world-positions",
		"make backfill-world-positions 补齐玩家世界地图权威坐标",
	} {
		if !strings.Contains(makefileContent, required) {
			t.Fatalf("Makefile missing %q", required)
		}
	}
}
