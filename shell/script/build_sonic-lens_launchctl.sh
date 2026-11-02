#!/bin/zsh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

echo "==> 工作目录: $PROJECT_ROOT"

# 1. 检查 media-control 依赖
if ! command -v media-control &> /dev/null; then
  echo "media-control 未安装，正在使用 Homebrew 安装..."
  if ! command -v brew &> /dev/null; then
    echo "错误：未找到 Homebrew，请先安装 Homebrew。"
    exit 1
  fi
  brew install media-control
else
  echo "media-control 已安装"
fi

# 2. 定位 Go 编译器（优先 asdf 活跃版本）
GO_BIN=""
if [[ -x "$HOME/.asdf/installs/golang/1.27.1/go/bin/go" ]]; then
  GO_BIN="$HOME/.asdf/installs/golang/1.27.1/go/bin/go"
elif command -v asdf &> /dev/null && asdf which go &> /dev/null; then
  GO_BIN="$(asdf which go)"
else
  GO_BIN="$(command -v go 2>/dev/null || true)"
fi

if [[ -z "$GO_BIN" || ! -x "$GO_BIN" ]]; then
  echo "错误：未找到可执行的 go 编译器"
  exit 1
fi

echo "==> 使用 Go 编译器: $GO_BIN ($("$GO_BIN" version))"

# 3. 编译 Go 二进制到 ./shell/bin/
mkdir -p ./shell/bin
mkdir -p ./.logs
echo "==> 正在编译 sonic-lens..."
"$GO_BIN" build -o ./shell/bin/sonic-lens .
chmod +x ./shell/bin/sonic-lens
echo "==> 编译完成: ./shell/bin/sonic-lens"

# 4. 同步 LaunchAgent 配置文件（动态替换路径占位符）
TARGET_PLIST="$HOME/Library/LaunchAgents/com.vincent.sonic-lens.job.plist"
SOURCE_PLIST="./shell/launch/com.vincent.sonic-lens.job.plist"
TMP_PLIST="$(mktemp /tmp/com.vincent.sonic-lens.job.plist.XXXXXX)"

echo "==> 渲染 plist 模板并同步到 $TARGET_PLIST"
sed "s|__PROJECT_ROOT__|$PROJECT_ROOT|g" "$SOURCE_PLIST" > "$TMP_PLIST"

if ! cp -f "$TMP_PLIST" "$TARGET_PLIST" 2>/dev/null; then
  echo "提示：由于历史文件权限原因需要 sudo 覆盖一次并修正所有者归还给当前用户..."
  sudo cp -f "$TMP_PLIST" "$TARGET_PLIST"
  sudo chown $(id -un):$(id -gn) "$TARGET_PLIST"
fi
rm -f "$TMP_PLIST"
chmod 644 "$TARGET_PLIST"

# 5. 重启 launchctl 守护服务
echo "==> 重新加载 launchctl 服务..."
launchctl unload "$TARGET_PLIST" 2>/dev/null || true
launchctl load "$TARGET_PLIST"

echo "==> 部署完成！检查服务状态:"
sleep 1
launchctl list | grep -E "sonic-lens" || echo "提示：服务已托管至 launchd"
