// 本文件归口数据库维护工具的配置读取。
package main

import (
	"fmt"

	"hero3/internal/platform/config"
)

// configuredDSN 读取当前数据库 DSN。
func configuredDSN() (string, error) {
	cfg := config.Load()
	if cfg.DatabaseDSN == "" {
		return "", fmt.Errorf("HERO3_DATABASE_DSN 未配置")
	}
	return cfg.DatabaseDSN, nil
}
