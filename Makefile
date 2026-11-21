.DEFAULT_GOAL := help
.PHONY: help init backend start stop restart reload status logs macos dmg install-macos client-install ios ipad test test-go test-bridge ui-snapshots ui-verify-ios clean

# 自动探测 Go 编译器（优先 PATH，其次 asdf 活跃版本）
GO_BIN := $(shell command -v go 2>/dev/null)
ifeq ($(GO_BIN),)
  ifneq ($(wildcard $(HOME)/.asdf/installs/golang/1.27.1/go/bin/go),)
    GO_BIN := $(HOME)/.asdf/installs/golang/1.27.1/go/bin/go
  else ifneq ($(wildcard $(HOME)/.asdf/shims/go),)
    GO_BIN := $(HOME)/.asdf/shims/go
  else
    GO_BIN := go
  endif
endif

help:
	@echo ""
	@echo "🎵 sonic-lens 统一构建、服务与打包代理中枢 (Makefile)"
	@echo "=================================================================="
	@echo "⚙️ Go 后端服务 (launchd 代理):"
	@echo "  make init            编译后端并安装/重载 launchd 守护服务"
	@echo "  make start           启动 launchd 托管的 sonic-lens 服务"
	@echo "  make stop            停止 launchd 服务"
	@echo "  make restart         停止、重新编译最新 Go 后端并启动"
	@echo "  make reload          不编译直接重启已有二进制服务"
	@echo "  make status          查看 launchd 任务与 8082 端口状态"
	@echo "  make logs            实时追踪服务 stdout/stderr 日志"
	@echo ""
	@echo "🍏 macOS 客户端 (Soniclens Bridge):"
	@echo "  make macos           仅编译 Release 版 Soniclens Bridge.app"
	@echo "  make dmg             完整打包 macOS DMG + ZIP + SHA256 (dist/)"
	@echo "  make install-macos   编译并覆盖安装至 /Applications 并重启客户端"
	@echo ""
	@echo "📱 iOS / iPadOS 客户端:"
	@echo "  make ios             编译验证 SoniclensBridgePhone (generic/platform=iOS)"
	@echo "  make ipad            编译验证 SoniclensBridgePad (generic/platform=iOS)"
	@echo ""
	@echo "🧪 自动化测试 & 清理:"
	@echo "  make test            运行 Go 全量单元测试"
	@echo "  make test-bridge     运行 Swift 客户端单元与快照测试 (Mac + Phone)"
	@echo "  make ui-snapshots    运行 macOS SwiftUI 离屏快照自检并输出 PNG"
	@echo "  make ui-verify-ios   启动 iPhone 模拟器加载 Mock 数据并截取画面"
	@echo "  make clean           清理 dist/ 与 .derivedData/ 构建缓存"
	@echo "=================================================================="
	@echo ""

# ==============================================================================
# Go 后端服务管理（代理调度 sonic-lens-run.sh）
# ==============================================================================
init:
	@zsh ./sonic-lens-run.sh init

backend: init

start:
	@zsh ./sonic-lens-run.sh start

stop:
	@zsh ./sonic-lens-run.sh stop

restart:
	@zsh ./sonic-lens-run.sh restart

reload:
	@zsh ./sonic-lens-run.sh reload

status:
	@zsh ./sonic-lens-run.sh status

logs:
	@zsh ./sonic-lens-run.sh logs

# ==============================================================================
# macOS 客户端构建与打包（代理调度 package_macos.sh / sonic-lens-run.sh）
# ==============================================================================
macos:
	@bash ./soniclens-bridge/scripts/package_macos.sh --app-only

dmg:
	@bash ./soniclens-bridge/scripts/package_macos.sh

install-macos:
	@zsh ./sonic-lens-run.sh client-install

client-install: install-macos

# ==============================================================================
# iOS / iPadOS 客户端构建验证
# ==============================================================================
ios:
	@xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj \
		-scheme SoniclensBridgePhone \
		-configuration Release \
		-destination 'generic/platform=iOS' \
		-derivedDataPath .derivedData/iOS_release \
		CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED=NO build -quiet

ipad:
	@xcodebuild -project soniclens-bridge/SoniclensBridge.xcodeproj \
		-scheme SoniclensBridgePad \
		-configuration Release \
		-destination 'generic/platform=iOS' \
		-derivedDataPath .derivedData/iPadOS_release \
		CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED=NO build -quiet

# ==============================================================================
# 测试与清理
# ==============================================================================
test-go:
	@"$(GO_BIN)" test ./...

test: test-go

ui-snapshots:
	@bash ./soniclens-bridge/scripts/verify_ui.sh snapshots

ui-verify-ios:
	@bash ./soniclens-bridge/scripts/verify_ui.sh ios-sim

test-bridge:
	@bash ./soniclens-bridge/scripts/verify_ui.sh snapshots
	@bash ./soniclens-bridge/scripts/verify_ui.sh phone-tests

clean:
	@echo "🧹 清理 sonic-lens 构建产物与缓存..."
	@rm -rf dist .derivedData
	@echo "✨ 清理完成"
