#!/bin/zsh

PLIST="$HOME/Library/LaunchAgents/com.vincent.sonic-lens.job.plist"
if [[ ! -f "$PLIST" ]]; then
  echo "未检测到 $PLIST，请先运行 ./shell/script/build_sonic-lens_launchctl.sh 进行初始化配置。"
  exit 1
fi

echo "正在加载并启动 com.vincent.sonic-lens.job ..."
launchctl unload "$PLIST" 2>/dev/null || true
launchctl load "$PLIST"
sleep 1
echo "已触发启动。检查监听端口 8082:"
lsof -i :8082 || echo "服务正在拉起中，请稍候检查..."
