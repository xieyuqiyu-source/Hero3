// 本文件验证 Hero3 服务配置的环境变量默认值。
package config

import "testing"

func TestLoadStartupMigrationsDefaultByEnvironment(t *testing.T) {
	t.Setenv("HERO3_ENV", "production")
	t.Setenv("HERO3_RUN_STARTUP_MIGRATIONS", "")
	if cfg := Load(); cfg.RunStartupMigrations {
		t.Fatal("expected production to skip startup migrations by default")
	}

	t.Setenv("HERO3_ENV", "development")
	t.Setenv("HERO3_RUN_STARTUP_MIGRATIONS", "")
	if cfg := Load(); !cfg.RunStartupMigrations {
		t.Fatal("expected development to run startup migrations by default")
	}
}

func TestLoadStartupMigrationsAllowsExplicitOverride(t *testing.T) {
	t.Setenv("HERO3_ENV", "production")
	t.Setenv("HERO3_RUN_STARTUP_MIGRATIONS", "true")
	if cfg := Load(); !cfg.RunStartupMigrations {
		t.Fatal("expected explicit true to enable startup migrations")
	}

	t.Setenv("HERO3_ENV", "development")
	t.Setenv("HERO3_RUN_STARTUP_MIGRATIONS", "false")
	if cfg := Load(); cfg.RunStartupMigrations {
		t.Fatal("expected explicit false to disable startup migrations")
	}
}
