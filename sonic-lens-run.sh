#!/bin/zsh

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_DIR="$PROJECT_ROOT/shell/script"
PLIST="$HOME/Library/LaunchAgents/com.vincent.sonic-lens.job.plist"
CLIENT_PACKAGE_SCRIPT="$PROJECT_ROOT/soniclens-bridge/scripts/package_macos.sh"
DIST_APP="$PROJECT_ROOT/dist/Soniclens Bridge.app"
DEST_APP="/Applications/Soniclens Bridge.app"

case "$1" in
  init|build)
    zsh "$SCRIPT_DIR/build_sonic-lens_launchctl.sh"
    ;;
  start)
    zsh "$SCRIPT_DIR/start_sonic-lens.sh"
    ;;
  stop)
    shift
    zsh "$SCRIPT_DIR/stop_sonic-lens.sh" "$@"
    ;;
  restart)
    echo "==> 正在停止服务并重新编译最新代码..."
    zsh "$SCRIPT_DIR/stop_sonic-lens.sh"
    sleep 1
    zsh "$SCRIPT_DIR/build_sonic-lens_launchctl.sh"
    ;;
  reload)
    echo "==> 正在重新加载现有服务 (不重新编译)..."
    zsh "$SCRIPT_DIR/stop_sonic-lens.sh"
    sleep 1
    zsh "$SCRIPT_DIR/start_sonic-lens.sh"
    ;;
  status)
    echo "==> launchd 任务状态:"
    launchctl print gui/$(id -u)/com.vincent.sonic-lens.job 2>/dev/null | grep -E "state =|pid =|last exit code =" || echo "未在运行或未注册"
    echo "\n==> 端口 8082 监听状态:"
    lsof -i :8082 || echo "8082 端口未在监听"
    ;;
  logs|log)
    tail -f "$PROJECT_ROOT/.logs/launchd_stdout.log" "$PROJECT_ROOT/.logs/launchd_stderr.log"
    ;;
  client-install|client|install-client)
    echo "=========================================="
    echo "🚀 开始构建并替换安装 macOS 客户端..."
    echo "=========================================="
    # 1. 构建最新 App
    bash "$CLIENT_PACKAGE_SCRIPT" --app-only

    if [[ ! -d "$DIST_APP" ]]; then
      echo "❌ 未找到构建产物: $DIST_APP"
      exit 1
    fi

    # 2. 优雅退出正在运行的客户端
    echo "==> 正在检查并退出已有客户端进程..."
    pkill -x "Soniclens Bridge" 2>/dev/null || true
    pkill -x "Soniclens" 2>/dev/null || true
    sleep 1

    # 3. 覆盖安装到 /Applications
    echo "==> 正在替换安装到 $DEST_APP ..."
    rm -rf "$DEST_APP"
    cp -R "$DIST_APP" "/Applications/"

    # 4. 重新拉起应用
    echo "==> 重新启动客户端..."
    open -a "$DEST_APP"
    echo "🎉 客户端最新版本已成功安装并启动！"
    ;;
  client-pack|package-client)
    echo "=========================================="
    echo "📦 正在执行完整打包 (包含 DMG 与 ZIP)..."
    echo "=========================================="
    shift
    bash "$CLIENT_PACKAGE_SCRIPT" "$@"
    ;;
  *)
    echo "用法: $0 {init|start|stop|restart|reload|status|logs|client-install|client-pack}"
    echo "  --- 服务端 (Backend) ---"
    echo "  init           : 编译项目并安装/重新加载 launchd 守护任务"
    echo "  start          : 启动 launchd 托管服务"
    echo "  stop           : 停止服务 (加 --uninstall 可彻底清理 launchd 配置与二进制)"
    echo "  restart        : 停止服务，重新编译最新代码并启动"
    echo "  reload         : 仅重启已有二进制 (不重新编译)"
    echo "  status         : 查看当前服务端与端口运行状态"
    echo "  logs           : 查看标准输出与标准错误日志"
    echo ""
    echo "  --- 客户端 (Client / SwiftUI) ---"
    echo "  client-install : 打包最新客户端代码并直接覆盖安装到 /Applications 并重启 (别名: client)"
    echo "  client-pack    : 完整打包客户端并生成 DMG/ZIP 发布包 (别名: package-client)"
    exit 1
    ;;
esac
