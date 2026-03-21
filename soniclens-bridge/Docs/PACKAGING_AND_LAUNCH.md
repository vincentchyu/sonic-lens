# Soniclens Bridge 打包与启动指南

本文档说明 `soniclens-bridge` 在本地开发机上的环境准备、macOS 打包启动，以及 iPad 真机与 iPad 模拟器的打包启动流程。

## 1. 环境准备

### 1.1 基础要求
- macOS 开发机
- Xcode 已安装，建议使用较新的正式版本
- Command Line Tools 可用
- 已安装 [XcodeGen](https://github.com/yonaskolb/XcodeGen)
- 如需真机调试，已登录 Apple ID，并在 Xcode 中配置可用 Team
- SonicLens 服务端与调试设备在同一局域网

### 1.2 关键目录
- 工程目录：`/Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge`
- Xcode 工程：`/Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensBridge.xcodeproj`
- iPad App 入口：`/Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensBridge/SoniclensBridgePadApp.swift`
- macOS App 入口：`/Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge/SoniclensBridge/SoniclensBridgeApp.swift`

### 1.3 安装与检查命令
```bash
# 检查 Xcode
xcodebuild -version

# 检查命令行工具
xcode-select -p

# 安装 XcodeGen（如未安装）
brew install xcodegen

# 生成工程
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge
xcodegen generate
```

### 1.4 iPad 模拟器运行时准备
如果 `xcodebuild -showdestinations` 看不到 iPad Simulator，通常是没有安装 iOS Simulator runtime。

```bash
# 下载 iOS 平台运行时
xcodebuild -downloadPlatform iOS

# 检查已安装 runtime
xcrun simctl list runtimes

# 检查可用 iPad 设备类型
xcrun simctl list devicetypes | rg iPad

# 检查当前可用模拟器
xcrun simctl list devices available
```

如果需要创建与真机接近的模拟器，例如 `iPad Air (5th generation)`：

```bash
xcrun simctl create "iPad Air 5" \
  com.apple.CoreSimulator.SimDeviceType.iPad-Air-5th-generation \
  com.apple.CoreSimulator.SimRuntime.iOS-26-1
```

### 1.5 本项目当前 Scheme
- macOS：`SoniclensBridgeMac`
- iPadOS：`SoniclensBridgePad`

### 1.6 Bonjour 与局域网要求
Bridge 依赖局域网发现 SonicLens 服务端。当前 Info.plist 已配置：
- `NSLocalNetworkUsageDescription`
- `NSBonjourServices = _soniclens._tcp`

建议先确认服务端健康检查正常：

```bash
curl http://<server-ip>:8082/health
```

## 2. macOS 打包与启动

### 2.1 使用 Xcode 启动 macOS 版本
1. 打开 `SoniclensBridge.xcodeproj`。
2. 选择 Scheme：`SoniclensBridgeMac`。
3. 运行目标选择 `My Mac`。
4. 执行 `Product -> Run`。

### 2.2 命令行构建 macOS 版本
```bash
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge

xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgeMac \
  -destination 'generic/platform=macOS' \
  -derivedDataPath "../.derivedData/macOS" build 2>&1 | tail -20
```

### 2.3 命令行定位产物
```bash
xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgeMac \
  -destination 'generic/platform=macOS' \
  -showBuildSettings | rg 'BUILT_PRODUCTS_DIR|FULL_PRODUCT_NAME'
```

### 2.4 Archive 与导出
1. 在 Xcode 中选择 `SoniclensBridgeMac`。
2. 执行 `Product -> Archive`。
3. 在 Organizer 中选择导出方式。

本地测试通常使用：
- `Development`
- `Copy App`

### 2.5 macOS 启动验证清单
- App 可以正常打开
- 连接页可以通过 Bonjour 发现服务端
- 手动输入 `IP:Port` 可以连通
- 首页统计、资料库、Mini Player、Now Playing 可以正常加载

## 3. iPad 真机打包与启动

### 3.1 Xcode 真机运行
1. 用 Xcode 打开工程。
2. 选择 Scheme：`SoniclensBridgePad`。
3. 在 `Signing & Capabilities` 中选择可用 Team。
4. 将真机连接到 Mac。
5. 在运行目标中选择你的 iPad。
6. 执行 `Product -> Run`。

### 3.2 真机常见前置检查
- iPad 已信任开发者证书
- 设备系统版本高于项目最低版本要求
- Bundle Identifier 不与现有安装包冲突
- Team、签名证书、Provisioning Profile 匹配

### 3.3 命令行真机构建
真机构建通常只用于验证签名和编译，不直接从命令行安装：

```bash
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge

xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -destination 'generic/platform=iOS' \
  build
```

### 3.4 Archive 与导出 iPad 包
1. 选择 Scheme：`SoniclensBridgePad`
2. 选择目标：`Any iOS Device (arm64)` 或通用 iOS 设备
3. 执行 `Product -> Archive`
4. 在 Organizer 中选择导出：

常见导出方式：
- `Development`
- `Ad Hoc`
- `App Store Connect`

## 4. iPad 模拟器打包与启动

### 4.1 查看可用 destination
```bash
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens/soniclens-bridge

xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -showdestinations
```

### 4.2 构建 iPad 模拟器包
以下命令以 `iPad Air 5` 为例：

```bash
xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -destination "platform=iOS Simulator,name=iPad Air 5,OS=26.1" \
  -derivedDataPath "../.derivedData/iPad" \
  clean build
```

### 4.3 启动模拟器
```bash
open -a Simulator
xcrun simctl boot "iPad Air 5"
```

如果设备已经启动，`boot` 失败可以忽略。

### 4.4 安装并启动 App
```bash
xcrun simctl install "iPad Air 5" \
  ../.derivedData/iPad/Build/Products/Debug-iphonesimulator/SoniclensBridgePad.app

xcrun simctl launch "iPad Air 5" com.vincentchyu.soniclens-bridge.pad
```

如果你希望用确定路径，先查看最新产物目录：

```bash
xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -destination "platform=iOS Simulator,name=iPad Air 5,OS=26.1" \
  -showBuildSettings | rg 'BUILT_PRODUCTS_DIR|FULL_PRODUCT_NAME'
```

### 4.5 卸载旧包
当 Bundle 被旧构建污染，或者切换过 `Info.plist` / launch screen / 签名配置时，建议先卸载后重装：

```bash
xcrun simctl uninstall "iPad Air 5" com.vincentchyu.soniclens-bridge.pad
xcrun simctl install "iPad Air 5" \
  ../.derivedData/iPad/Build/Products/Debug-iphonesimulator/SoniclensBridgePad.app
```

### 4.6 查看运行日志
当 `simctl launch` 失败时，优先看 SpringBoard 和 App 本身日志：

```bash
xcrun simctl spawn "iPad Air 5" log show \
  --style compact \
  --last 3m \
  --predicate 'process == "SpringBoard" OR process == "SoniclensBridgePad"'
```

如果需要实时看日志：

```bash
xcrun simctl spawn "iPad Air 5" log stream \
  --style compact \
  --level debug \
  --predicate 'process == "SpringBoard" OR process == "SoniclensBridgePad"'
```

## 5. 常见问题

### 5.1 `xcodebuild -showdestinations` 看不到 iPad
通常原因：
- 没安装 iOS Simulator runtime
- 当前 Xcode 未完成首次初始化
- Scheme 不是 iOS 目标

排查命令：

```bash
xcodebuild -downloadPlatform iOS
xcrun simctl list runtimes
xcodebuild -project SoniclensBridge.xcodeproj -scheme SoniclensBridgePad -showdestinations
```

### 5.2 `simctl launch` 提示 `SBMainWorkspace` 拒绝启动
优先排查：
- app bundle 里是否有错误的资源污染
- launch screen 是否有效
- 是否安装了旧包

建议执行：

```bash
xcodegen generate

xcodebuild \
  -project SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -destination "platform=iOS Simulator,name=iPad Air 5,OS=26.1" \
  clean build

xcrun simctl uninstall "iPad Air 5" com.vincentchyu.soniclens-bridge.pad
xcrun simctl install "iPad Air 5" \
  /Users/vincent/Library/Developer/Xcode/DerivedData/SoniclensBridge-ckmdhzeltzsgwxfvbwjnbjermmdx/Build/Products/Debug-iphonesimulator/SoniclensBridgePad.app
xcrun simctl launch "iPad Air 5" com.vincentchyu.soniclens-bridge.pad
```

### 5.3 Bonjour 发现不到服务端
- 确认 iPad / 模拟器与服务端处于同一网络环境
- 确认服务端已广播 `_soniclens._tcp`
- 确认 `/health` 正常返回
- 首次启动时允许本地网络权限

### 5.4 真机构建能编译但无法安装
通常是签名问题：
- Team 未配置
- Provisioning Profile 不匹配
- Bundle Identifier 与已安装包冲突
- 设备未信任开发者

## 6. 推荐操作顺序

### 6.1 macOS 本地开发
1. `xcodegen generate`
2. 选择 `SoniclensBridgeMac`
3. `Product -> Run`

### 6.2 iPad 模拟器开发
1. 安装 iOS Simulator runtime
2. 创建或选择 iPad 模拟器
3. `xcodegen generate`
4. `xcodebuild ... clean build`
5. `simctl install`
6. `simctl launch`

### 6.3 iPad 真机联调
1. 配置 Team
2. 真机连接 Xcode
3. 选择 `SoniclensBridgePad`
4. `Product -> Run`
