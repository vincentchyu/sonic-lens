# 构建与验证指南（macOS）

本指南说明如何在 macOS 上构建并验证 **Soniclens Bridge**。

## 前置条件
- macOS + 已安装 Xcode（建议 Xcode 15+）
- Apple Developer Team 已配置
- SonicLens 服务端与本机在同一局域网

## 项目位置
- Xcode 工程：`./soniclens-bridge/SoniclensBridge.xcodeproj`

## Xcode 构建与运行（推荐）
1. 使用 Xcode 打开工程。
2. 选择 Target：`SoniclensBridge`。
3. Signing & Capabilities：
   - Team：选择你的 Apple ID Team。
   - Bundle ID：默认 `com.vincentchyu.soniclens-bridge`（需要可自行修改）。
4. 选择运行目标：
   - iOS 模拟器，或
   - **My Mac (Mac Catalyst)**。
5. 执行 `Product -> Run`。

## 命令行构建
```bash
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge

# iOS 模拟器构建
xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridge \
  -sdk iphonesimulator \
  -configuration Debug \
  build

# Mac Catalyst 构建
xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridge \
  -configuration Debug \
  -destination 'platform=macOS,variant=Mac Catalyst' \
  build
```

## Archive 与导出
1. Xcode -> `Product -> Archive`。
2. 在 Organizer 中选择导出方式：
   - 本地验证：**Development** 或 **Ad Hoc**。

## 验证清单
- App 能打开连接页。
- Bonjour 自动发现能列出服务端（若服务端已广播）。
- 手动输入 IP:Port 可连接。
- 主页能加载：统计 / 趋势 / 排行 / 最近播放。
- 资料库能加载：专辑 / 曲目 / 音眸 / 未上报。
- Mini Player 能显示 Now Playing。
- 全屏播放页能展示歌词与音眸内容。

## Bonjour 与局域网说明
- App 的 Info.plist 已包含：
  - `NSLocalNetworkUsageDescription`
  - `NSBonjourServices: _soniclens._tcp`
- 服务端日志应出现：
  - `Bonjour 广播已启动`（端口需与 App 连接端口一致，例如 `8082`）

## 常见问题
- Bonjour 发现失败：
  - 确认客户端与服务端在同一局域网。
  - 检查服务端日志是否显示 Bonjour 已启动。
- API 请求失败：
  - 确认连接页端口与服务端端口一致。
  - 检查 `/health` 返回 `{"status":"ok"}`。
