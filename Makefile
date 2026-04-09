.PHONY: build build-backend build-frontend build-datamanagementd test test-backend test-frontend test-datamanagementd secret-scan \
	stable-up stable-down stable-logs stable-rebuild \
	dev-up dev-down dev-logs dev-rebuild \
	sync-upstream backup-runtime

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 编译 datamanagementd（宿主机数据管理进程）
build-datamanagementd:
	@cd datamanagement && go build -o datamanagementd ./cmd/datamanagementd

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck

test-datamanagementd:
	@cd datamanagement && go test ./...

secret-scan:
	@python3 tools/secret_scan.py

stable-up:
	@./scripts/sub2api-local stable up

stable-down:
	@./scripts/sub2api-local stable down

stable-logs:
	@./scripts/sub2api-local stable logs

stable-rebuild:
	@./scripts/sub2api-local stable rebuild

dev-up:
	@./scripts/sub2api-local dev up --build

dev-down:
	@./scripts/sub2api-local dev down

dev-logs:
	@./scripts/sub2api-local dev logs

dev-rebuild:
	@./scripts/sub2api-local dev rebuild

sync-upstream:
	@./scripts/sub2api-local sync upstream

backup-runtime:
	@./scripts/sub2api-local backup runtime
