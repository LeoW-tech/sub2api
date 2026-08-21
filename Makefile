.PHONY: build build-backend build-frontend test test-backend test-frontend test-frontend-critical secret-scan migration-check \
	stable-up stable-down stable-logs stable-status stable-restart stable-rebuild \
	dev-up dev-down dev-logs dev-status dev-restart dev-rebuild \
	door-restart door-status \
	systemd-install systemd-status systemd-restart \
	autostart-install autostart-uninstall autostart-status autostart-restart \
	sync-upstream backup-runtime

PNPM := $(shell if command -v pnpm >/dev/null 2>&1; then printf 'pnpm'; else printf 'corepack pnpm'; fi)

FRONTEND_CRITICAL_VITEST := \
	src/api/__tests__/client.spec.ts \
	src/api/__tests__/tokenRefresh.spec.ts \
	src/api/__tests__/channelMonitorV2.spec.ts \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/views/user/__tests__/ChannelStatusView.mode.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts \
	src/features/channel-monitor-v2/__tests__/designSystem.structure.spec.ts \
	src/features/channel-monitor-v2/__tests__/monitorFormat.spec.ts \
	src/features/channel-monitor-v2/__tests__/monitorZoom.spec.ts

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@$(PNPM) --dir frontend run build

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@$(PNPM) --dir frontend run lint:check
	@$(PNPM) --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-frontend-critical:
	@$(PNPM) --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)

secret-scan:
	@python3 tools/secret_scan.py

migration-check:
	@./scripts/check-migration-numbering
	@./scripts/tests/check-migration-numbering-test.sh

stable-up:
	@./scripts/sub2api-local stable up

stable-down:
	@./scripts/sub2api-local stable down

stable-logs:
	@./scripts/sub2api-local stable logs

stable-status:
	@./scripts/sub2api-local stable status

stable-restart:
	@./scripts/sub2api-local stable restart

stable-rebuild:
	@./scripts/sub2api-local stable rebuild

dev-up:
	@./scripts/sub2api-local dev up --build

dev-down:
	@./scripts/sub2api-local dev down

dev-logs:
	@./scripts/sub2api-local dev logs

dev-status:
	@./scripts/sub2api-local dev status

dev-restart:
	@./scripts/sub2api-local dev restart

dev-rebuild:
	@./scripts/sub2api-local dev rebuild

door-restart:
	@./scripts/sub2api-local door restart

door-status:
	@./scripts/sub2api-local door status

systemd-install:
	@./scripts/sub2api-local systemd install

systemd-status:
	@./scripts/sub2api-local systemd status

systemd-restart:
	@./scripts/sub2api-local systemd restart

autostart-install:
	@./scripts/sub2api-local autostart install

autostart-uninstall:
	@./scripts/sub2api-local autostart uninstall

autostart-status:
	@./scripts/sub2api-local autostart status

autostart-restart:
	@./scripts/sub2api-local autostart restart

sync-upstream:
	@./scripts/sub2api-local sync upstream

backup-runtime:
	@./scripts/sub2api-local backup runtime
