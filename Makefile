# Hero3 项目 Makefile
# 统一开发环境启动、构建、部署命令

.PHONY: dev dev-go dev-web dev-admin build build-go build-dbtool build-web build-admin clean install migrate migrate-test print-test-dsn clone-data cleanup-battle-reports-dry-run cleanup-battle-reports report-stats lock-snapshot ensure-report-cleanup-indexes-dry-run ensure-report-cleanup-indexes maintenance-status backfill-resources verify-resources backfill-inventory verify-inventory backfill-buildings verify-buildings backfill-resource-slots verify-resource-slots backfill-army verify-army verify-recruit-queues backfill-recruit-queues backfill-generals verify-generals backfill-buffs verify-buffs backfill-currencies verify-currencies backfill-npc-states verify-npc-states backfill-world-positions healthcheck-authority openapi openapi-lint openapi-bundle

# ===== 开发 =====

## 启动所有服务（Go 后端 + Web 前端 + GM 后台）
dev:
	./dev.sh

## 仅启动 Go 后端
dev-go:
	cd go && go run ./cmd/server

## 仅启动 Web 前端
dev-web:
	cd web && pnpm dev

## 仅启动 GM 后台
dev-admin:
	cd admin && pnpm dev

# ===== 安装依赖 =====

## 安装所有依赖
install:
	cd go && go mod download
	cd web && pnpm install
	cd admin && pnpm install

# ===== 构建 =====

## 构建所有
build: build-go build-web build-admin

## 构建 Go 后端
build-go:
	cd go && go build -o bin/server ./cmd/server

## 构建数据库维护工具
build-dbtool:
	cd go && go build -o bin/dbtool ./cmd/dbtool

## 构建 Web 前端
build-web:
	cd web && pnpm build

## 构建 GM 后台
build-admin:
	cd admin && pnpm build

# ===== 清理 =====

## 清理构建产物
clean:
	rm -rf go/bin
	rm -rf web/dist
	rm -rf admin/dist

# ===== 数据库 =====

## 运行数据库迁移
migrate:
	cd go && go run ./cmd/dbtool migrate

## 创建并迁移 test_ 前缀测试库
migrate-test:
	cd go && go run ./cmd/dbtool migrate-test

## 输出 test_ 前缀测试库 DSN（默认隐藏密码）
print-test-dsn:
	cd go && go run ./cmd/dbtool print-test-dsn

## 从 HERO3_SOURCE_DATABASE_DSN 复制数据到当前 test_ 目标库
clone-data:
	cd go && go run ./cmd/dbtool clone-data --truncate

## 从 state_json 回填 player_resources 权威表
backfill-resources:
	cd go && go run ./cmd/dbtool backfill-resources

## 校验 player_resources 与 state_json.resources 是否一致
verify-resources:
	cd go && go run ./cmd/dbtool verify-resources

## 从 state_json 回填 player_inventory 权威表
backfill-inventory:
	cd go && go run ./cmd/dbtool backfill-inventory

## 校验 player_inventory 与 state_json.inventory 是否一致
verify-inventory:
	cd go && go run ./cmd/dbtool verify-inventory

## 从 state_json 回填 player_buildings 权威表
backfill-buildings:
	cd go && go run ./cmd/dbtool backfill-buildings

## 校验 player_buildings 与 state_json.buildings 是否一致
verify-buildings:
	cd go && go run ./cmd/dbtool verify-buildings

## 从建筑快照回填 player_resource_slots 权威表
backfill-resource-slots:
	cd go && go run ./cmd/dbtool backfill-resource-slots

## 校验 player_resource_slots 与兼容快照是否一致
verify-resource-slots:
	cd go && go run ./cmd/dbtool verify-resource-slots

## 从 state_json 回填 player_army_units 权威表
backfill-army:
	cd go && go run ./cmd/dbtool backfill-army

## 校验 player_army_units 与 state_json.army 是否一致
verify-army:
	cd go && go run ./cmd/dbtool verify-army

## 从 state_json 回填 player_recruit_queues 权威表
backfill-recruit-queues:
	cd go && go run ./cmd/dbtool backfill-recruit-queues

## 校验 player_recruit_queues 与 state_json.recruitQueues 是否一致
verify-recruit-queues:
	cd go && go run ./cmd/dbtool verify-recruit-queues

## 从 state_json 回填 player_generals 和 player_general_assignments 权威表
backfill-generals:
	cd go && go run ./cmd/dbtool backfill-generals

## 校验 player_generals / player_general_assignments 与兼容快照是否一致
verify-generals:
	cd go && go run ./cmd/dbtool verify-generals

## 从 state_json 回填 player_buffs 权威表
backfill-buffs:
	cd go && go run ./cmd/dbtool backfill-buffs

## 校验 player_buffs 与 state_json.buffs 是否一致
verify-buffs:
	cd go && go run ./cmd/dbtool verify-buffs

## 从 state_json 补齐 player_currencies 权威表
backfill-currencies:
	cd go && go run ./cmd/dbtool backfill-currencies

## 校验 player_currencies 是否覆盖所有玩家
verify-currencies:
	cd go && go run ./cmd/dbtool verify-currencies

## 从 state_json.npcState 补齐 player_npc_states 权威表
backfill-npc-states:
	cd go && go run ./cmd/dbtool backfill-npc-states

## 校验旧 NPC 快照是否已有 player_npc_states 承接
verify-npc-states:
	cd go && go run ./cmd/dbtool verify-npc-states

## 为已有玩家补齐世界地图权威坐标
backfill-world-positions:
	cd go && go run ./cmd/dbtool backfill-world-positions

## 检查当前权威表覆盖和 state_json 轻量化状态
healthcheck-authority:
	cd go && go run ./cmd/dbtool healthcheck-authority

## dry-run 统计可清理战报
cleanup-battle-reports-dry-run:
	cd go && go run ./cmd/dbtool cleanup-battle-reports

## 正式分批清理过期和软删战报；生产库执行需显式允许
cleanup-battle-reports:
	cd go && go run ./cmd/dbtool cleanup-battle-reports --execute --allow-non-test --batch-size 500 --max-batches 4 --retention-hours 72 --pvp-retention-hours 168 --defense-retention-hours 168 --scout-retention-hours 168 --deleted-retention-hours 24

## 统计战报总量、每日增长和玩家 Top
report-stats:
	cd go && go run ./cmd/dbtool report-stats

## 只读输出活跃连接、事务和锁等待快照
lock-snapshot:
	cd go && go run ./cmd/dbtool lock-snapshot --min-seconds 1 --limit 30

## dry-run 检查战报清理和可见上限索引
ensure-report-cleanup-indexes-dry-run:
	cd go && go run ./cmd/dbtool ensure-report-cleanup-indexes

## 正式创建缺失的战报生命周期索引；生产库执行需显式允许
ensure-report-cleanup-indexes:
	cd go && go run ./cmd/dbtool ensure-report-cleanup-indexes --execute --allow-non-test

## 只读汇总战报、清理索引和权威表健康状态
maintenance-status:
	cd go && go run ./cmd/dbtool maintenance-status

# ===== 接口文档 =====

## 校验拆分后的 OpenAPI 入口文件
openapi-lint:
	python3 scripts/openapi_bundle.py --input docs/接口文档/openapi/openapi.yaml --output docs/接口文档/openapi打包.yaml

## 打包 Apifox 导入文件
openapi-bundle: openapi-lint

## 校验并打包 OpenAPI
openapi: openapi-bundle

# ===== 帮助 =====

## 显示帮助
help:
	@echo "Hero3 开发命令："
	@echo ""
	@echo "  make dev          启动所有服务"
	@echo "  make dev-go       仅启动 Go 后端"
	@echo "  make dev-web      仅启动 Web 前端"
	@echo "  make dev-admin    仅启动 GM 后台"
	@echo ""
	@echo "  make install      安装所有依赖"
	@echo "  make build        构建所有"
	@echo "  make clean        清理构建产物"
	@echo "  make migrate      运行数据库迁移"
	@echo "  make backfill-world-positions 补齐玩家世界地图权威坐标"
	@echo "  make healthcheck-authority 检查权威表覆盖"
	@echo "  make report-stats 统计战报增长"
	@echo "  make lock-snapshot 查看数据库锁等待"
	@echo "  make ensure-report-cleanup-indexes-dry-run 检查战报清理索引"
	@echo "  make maintenance-status 汇总维护健康状态"
	@echo "  make openapi      校验并打包接口文档"
