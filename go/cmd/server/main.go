package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"hero3/internal/app/game"
	_ "hero3/internal/core/general/traits" // 触发将领特性自动注册
	"hero3/internal/infrastructure/storage"
	"hero3/internal/platform/config"
	"hero3/internal/platform/httpserver"
	"hero3/internal/transport/api"
)

func main() {
	// 自动加载 .env 文件（不存在则忽略）
	_ = godotenv.Load()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	gameService := game.NewService()
	if err := gameService.SetBalancePath(cfg.BalancePath); err != nil {
		logger.Error("balance config load failed", "path", cfg.BalancePath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetFactionsPath(cfg.FactionsPath); err != nil {
		logger.Error("factions config load failed", "path", cfg.FactionsPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetUnitsDir(cfg.UnitsDir); err != nil {
		logger.Error("units config load failed", "dir", cfg.UnitsDir, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetNpcConfigPath(cfg.NpcConfigPath); err != nil {
		logger.Error("npc config load failed", "path", cfg.NpcConfigPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetCombatPath(cfg.CombatPath); err != nil {
		logger.Error("combat config load failed", "path", cfg.CombatPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetGeneralsPath(cfg.GeneralsPath); err != nil {
		logger.Error("generals config load failed", "path", cfg.GeneralsPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetItemsPath(cfg.ItemsPath); err != nil {
		logger.Error("items config load failed", "path", cfg.ItemsPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetDropPoolsPath(cfg.DropPoolsPath); err != nil {
		logger.Error("drop pools config load failed", "path", cfg.DropPoolsPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetFishingPath(cfg.FishingPath); err != nil {
		logger.Error("fishing config load failed", "path", cfg.FishingPath, "error", err)
		os.Exit(1)
	}
	if err := gameService.SetReincarnationPath(cfg.ReincarnationPath); err != nil {
		logger.Error("reincarnation config load failed", "path", cfg.ReincarnationPath, "error", err)
		os.Exit(1)
	}
	if cfg.DatabaseDSN != "" {
		if err := validateDevelopmentDatabase(cfg); err != nil {
			logger.Error("database safety check failed", "error", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := storage.OpenMySQL(ctx, cfg.DatabaseDSN)
		if err != nil {
			logger.Error("database connection failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		if shouldRunStartupMigrations(cfg) {
			if err := storage.MigrateMySQL(ctx, db); err != nil {
				logger.Error("database migration failed", "error", err)
				os.Exit(1)
			}
		} else {
			logger.Info("database startup migration skipped", "env", cfg.Environment)
		}

		gameService = game.NewServiceWithRepository(storage.NewMySQLRepository(db))
		if err := gameService.SetBalancePath(cfg.BalancePath); err != nil {
			logger.Error("balance config load failed", "path", cfg.BalancePath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetFactionsPath(cfg.FactionsPath); err != nil {
			logger.Error("factions config load failed", "path", cfg.FactionsPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetUnitsDir(cfg.UnitsDir); err != nil {
			logger.Error("units config load failed", "dir", cfg.UnitsDir, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetNpcConfigPath(cfg.NpcConfigPath); err != nil {
			logger.Error("npc config load failed", "path", cfg.NpcConfigPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetCombatPath(cfg.CombatPath); err != nil {
			logger.Error("combat config load failed", "path", cfg.CombatPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetGeneralsPath(cfg.GeneralsPath); err != nil {
			logger.Error("generals config load failed", "path", cfg.GeneralsPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetItemsPath(cfg.ItemsPath); err != nil {
			logger.Error("items config load failed", "path", cfg.ItemsPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetDropPoolsPath(cfg.DropPoolsPath); err != nil {
			logger.Error("drop pools config load failed", "path", cfg.DropPoolsPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetFishingPath(cfg.FishingPath); err != nil {
			logger.Error("fishing config load failed", "path", cfg.FishingPath, "error", err)
			os.Exit(1)
		}
		if err := gameService.SetReincarnationPath(cfg.ReincarnationPath); err != nil {
			logger.Error("reincarnation config load failed", "path", cfg.ReincarnationPath, "error", err)
			os.Exit(1)
		}
		logger.Info("database storage enabled")
	} else {
		logger.Info("memory storage enabled")
	}

	router := api.NewRouter(api.RouterOptions{
		Config:      cfg,
		Logger:      logger,
		GameService: gameService,
	})

	server := httpserver.New(cfg, logger, router)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Hero3 API server starting", "addr", cfg.Addr, "env", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	case signal := <-shutdownSignals:
		logger.Info("shutdown signal received", "signal", signal.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Hero3 API server stopped")
}

// shouldRunStartupMigrations 判断服务启动时是否自动执行数据库结构迁移。
func shouldRunStartupMigrations(cfg config.Config) bool {
	return cfg.RunStartupMigrations
}

// validateDevelopmentDatabase 防止本地开发模式误连稳定玩家库。
func validateDevelopmentDatabase(cfg config.Config) error {
	if !strings.EqualFold(strings.TrimSpace(cfg.Environment), "development") || cfg.AllowDevDB {
		return nil
	}
	databaseName, err := storage.MySQLDatabaseName(cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	if strings.HasPrefix(databaseName, "test_") {
		return nil
	}
	return errors.New("development database must use test_ prefix or set HERO3_ALLOW_DEVELOPMENT_DATABASE=true")
}
