// 本文件验证服务启动时的开发库保护规则。
package main

import (
	"strings"
	"testing"

	"hero3/internal/platform/config"
)

func TestValidateDevelopmentDatabaseAllowsTestDatabase(t *testing.T) {
	cfg := config.Config{
		Environment: "development",
		DatabaseDSN: "hero3_user:hero3_password@tcp(127.0.0.1:3306)/test_hero3?parseTime=true",
	}

	if err := validateDevelopmentDatabase(cfg); err != nil {
		t.Fatalf("expected test database to pass, got %v", err)
	}
}

func TestValidateDevelopmentDatabaseRejectsStableDatabase(t *testing.T) {
	cfg := config.Config{
		Environment: "development",
		DatabaseDSN: "hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true",
	}

	err := validateDevelopmentDatabase(cfg)
	if err == nil {
		t.Fatal("expected stable database to be rejected in development")
	}
	if !strings.Contains(err.Error(), "test_ prefix") {
		t.Fatalf("expected test_ prefix error, got %v", err)
	}
}

func TestValidateDevelopmentDatabaseAllowsExplicitOverride(t *testing.T) {
	cfg := config.Config{
		Environment: "development",
		DatabaseDSN: "hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true",
		AllowDevDB:  true,
	}

	if err := validateDevelopmentDatabase(cfg); err != nil {
		t.Fatalf("expected explicit override to pass, got %v", err)
	}
}

func TestValidateDevelopmentDatabaseAllowsNonDevelopmentEnvironment(t *testing.T) {
	cfg := config.Config{
		Environment: "production",
		DatabaseDSN: "hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true",
	}

	if err := validateDevelopmentDatabase(cfg); err != nil {
		t.Fatalf("expected non-development environment to pass, got %v", err)
	}
}

func TestValidateDevelopmentDatabaseReturnsDSNError(t *testing.T) {
	cfg := config.Config{
		Environment: "development",
		DatabaseDSN: "not a mysql dsn",
	}

	if err := validateDevelopmentDatabase(cfg); err == nil {
		t.Fatal("expected invalid dsn to fail")
	}
}
