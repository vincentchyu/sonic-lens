#!/bin/zsh

PLIST="$HOME/Library/LaunchAgents/com.vincent.sonic-lens.job.plist"

echo "正在停止 com.vincent.sonic-lens.job ..."
if [[ -f "$PLIST" ]]; then
  launchctl unload "$PLIST" 2>/dev/null || true
fi
launchctl stop com.vincent.sonic-lens.job 2>/dev/null || true

if [[ "$1" == "--uninstall" ]]; then
  echo "正在清理 launchd 配置与本地二进制..."
  launchctl remove com.vincent.sonic-lens.job 2>/dev/null || true
  rm -f "$PLIST"
  rm -f ./shell/bin/sonic-lens
fi

echo "服务已停止。"
