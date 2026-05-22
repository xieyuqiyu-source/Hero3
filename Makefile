# Hero3 项目 Makefile
# 统一开发环境启动、构建、部署命令

.PHONY: dev dev-go dev-web dev-admin build build-go build-web build-admin clean install openapi openapi-lint openapi-bundle

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
	cd go && go run ./cmd/server migrate

# ===== 接口文档 =====

## 校验拆分后的 OpenAPI 入口文件
openapi-lint:
	python3 scripts/openapi_bundle.py --input docs/openapi/openapi.yaml --output docs/openapi.bundle.yaml

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
