# SonicLens Bridge 构建与验证指南

本指南说明如何构建并验证 `soniclens-bridge` 的 Mac / iPad / iPhone 三端目标，并补充当前性能治理后的重点回归场景。

## 前置条件

- macOS + 已安装 Xcode（建议 Xcode 15+）
- Apple Developer Team 已配置
- SonicLens 服务端与测试设备在同一局域网

## 项目位置

- Xcode 工程：`./soniclens-bridge/SoniclensBridge.xcodeproj`

## 当前主要 Scheme

- `SoniclensBridgeMac`
- `SoniclensBridgePad`
- `SoniclensBridgePhone`

## Xcode 构建与运行（推荐）

1. 使用 Xcode 打开工程。
2. 选择对应 Scheme：
   - macOS：`SoniclensBridgeMac`
   - iPad：`SoniclensBridgePad`
   - iPhone：`SoniclensBridgePhone`
3. Signing & Capabilities：
   - Team：选择你的 Apple ID Team。
   - 如需修改 Bundle ID，请同步检查 `project.yml` 中的生成配置。
4. 选择运行目标并执行 `Product -> Run`。

## 命令行构建

```bash
cd /Users/vincent/Developer/code/go_code/src/github.com/vincentchyu/sonic-lens

# macOS
xcodebuild \
  -project soniclens-bridge/SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgeMac \
  -configuration Debug \
  -sdk macosx \
  build \
  CODE_SIGNING_ALLOWED=NO

# iPhone
xcodebuild \
  -project soniclens-bridge/SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePhone \
  -configuration Debug \
  -destination 'generic/platform=iOS' \
  build \
  CODE_SIGNING_ALLOWED=NO
```

## 功能冒烟清单

- App 能打开连接页。
- Bonjour 自动发现能列出服务端。
- 手动输入 IP:Port 可连接。
- 连接成功后工具栏 `power` 按钮可断开。
- 主页能加载统计 / 趋势 / 排行 / 最近播放。
- 资料库能加载专辑 / 曲目 / 音眸 / 未上报。
- 播放条与全屏播放页能显示当前播放、歌词与音眸内容。

## 性能回归清单

### 连接链路

- 进入连接页后点击自动发现候选，应立即看到顶部与行内阶段反馈。
- 连接中再次点击同一目标应取消当前连接，而不是并发多条连接链路。
- Bonjour 候选如果带解析地址，健康检查耗时应明显低于重复走 `.local` 名称解析的旧链路。

### 资料库

- 首次进入专辑页后立刻打开排序。
- 首次进入曲目页后立刻打开筛选。
- 收藏筛选、未上报筛选首次进入不应出现明显整页卡顿。
- 快速切换排序 / 筛选 / 搜索时，旧结果不能覆盖新条件结果。

### 当前播放与首页

- 连续切歌时歌词 / insight 不应串行阻塞。
- 反复打开 / 关闭正在播放页不应明显掉帧。
- 首页常驻 1 到 2 分钟后切页再返回，动态背景不应明显抖动或升温。

### Instruments / 调试工具

- Hangs
- SwiftUI
- Time Profiler
- Core Animation
- Network

更细的长期性能约束见：
- `Docs/PERFORMANCE_GUARDRAILS.md`

## Bonjour 与局域网说明

- App 的 Info.plist 已包含：
  - `NSLocalNetworkUsageDescription`
  - `NSBonjourServices: _soniclens._tcp`
- 服务端日志应出现：
  - `Bonjour 广播已启动`
- 当前自动发现链路会保留显示用主机名和解析后的直连地址；连接时优先使用解析地址。

## 当前已知未闭环项

- iPhone 中文九宫格搜索输入在组合态下仍可能卡顿；此前实验性修复已回滚，后续处理需优先保证搜索框尺寸与布局不回归。

## 常见问题

- Bonjour 发现失败：
  - 确认客户端与服务端在同一局域网。
  - 检查服务端日志是否显示 Bonjour 已启动。
  - 检查 App 是否已授予“本地网络”权限。
- API 请求失败：
  - 确认连接页端口与服务端端口一致。
  - 检查 `/health` 是否返回 `{"status":"ok"}`。
