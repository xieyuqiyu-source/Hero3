// 本文件测试应用层高频入口不会回退到旧全量资产事务。
package game

import (
	"os"
	"strings"
	"testing"
)

// TestAppServicesDoNotCallLegacyFullAssetTransactions 防止服务层重新直接调用旧全量资产事务。
func TestAppServicesDoNotCallLegacyFullAssetTransactions(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read app game dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "repository.go" {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source := string(content)
		for _, forbidden := range []string{".UpdateItemState(", ".UpdateRewardState(", ".UpdateAccountRewardState("} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s must use scoped asset transactions instead of %s", name, forbidden)
			}
		}
	}
}
