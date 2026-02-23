#!/bin/zsh

# 检查 media-control 是否已安装
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

# 编译 Go 项目
go build
mv ./sonic-lens ./shell/bin/sonic-lens

# 启动代理服务
sudo cp ./shell/launch/com.vincent.sonic-lens.job.plist ~/Library/LaunchAgents/