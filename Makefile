# Hero3 项目 Makefile
# 统一开发环境启动、构建、部署命令

.PHONY: dev dev-go dev-web dev-admin build build-go build-web build-admin clean install migrate migrate-test print-test-dsn clone-data backfill-resources verify-resources backfill-inventory verify-inventory backfill-buildings verify-buildings backfill-resource-slots verify-resource-slots backfill-army verify-army backfill-recruit-queues verify-recruit-queues backfill-generals verify-generals backfill-buffs verify-buffs healthcheck-authority openapi openapi-lint openapi-bundle

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

## 检查当前权威表覆盖和 state_json 轻量化状态
healthcheck-authority:
	cd go && go run ./cmd/dbtool healthcheck-authority

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
	@echo "  make openapi      校验并打包接口文档"
