// 本文件验证数据库迁移命令的 DSN 处理逻辑。
package main

import (
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestNormalizeMigrationDSNRaisesShortTimeouts(t *testing.T) {
	dsn := "hero3:secret@tcp(127.0.0.1:3306)/hero3?parseTime=true&readTimeout=5s&writeTimeout=5s&timeout=5s"
	normalized, err := normalizeMigrationDSN(dsn)
	if err != nil {
		t.Fatalf("normalizeMigrationDSN failed: %v", err)
	}
	cfg, err := mysql.ParseDSN(normalized)
	if err != nil {
		t.Fatalf("ParseDSN failed: %v", err)
	}
	if cfg.Timeout < migrationConnTimeout {
		t.Fatalf("expected timeout >= %s, got %s", migrationConnTimeout, cfg.Timeout)
	}
	if cfg.ReadTimeout < migrationReadTimeout {
		t.Fatalf("expected read timeout >= %s, got %s", migrationReadTimeout, cfg.ReadTimeout)
	}
	if cfg.WriteTimeout < migrationReadTimeout {
		t.Fatalf("expected write timeout >= %s, got %s", migrationReadTimeout, cfg.WriteTimeout)
	}
}

func TestNormalizeMigrationDSNPreservesLongerTimeouts(t *testing.T) {
	dsn := "hero3:secret@tcp(127.0.0.1:3306)/hero3?parseTime=true&readTimeout=90s&writeTimeout=90s&timeout=45s"
	normalized, err := normalizeMigrationDSN(dsn)
	if err != nil {
		t.Fatalf("normalizeMigrationDSN failed: %v", err)
	}
	cfg, err := mysql.ParseDSN(normalized)
	if err != nil {
		t.Fatalf("ParseDSN failed: %v", err)
	}
	if cfg.Timeout != 45*time.Second {
		t.Fatalf("expected timeout preserved, got %s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 90*time.Second {
		t.Fatalf("expected read timeout preserved, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 90*time.Second {
		t.Fatalf("expected write timeout preserved, got %s", cfg.WriteTimeout)
	}
}
