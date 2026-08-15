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

## 客户端日志与沙盒调试指南

### 1. 日志机制
Bridge 客户端采用 Apple 统一日志系统（`os.Logger` / OSLog），日志不落裸文本文件，而是由 macOS / iOS `logd` 统一收集：
- **子系统标识**：`subsystem: "com.vincentchyu.soniclens-bridge"`
- **核心分类 (Category)**：
  - `APIClient`：HTTP 请求路径、入参、响应状态码与解码错误
  - `WebSocket`：正在播放状态推送、握手与心跳
  - `LibrarySync`：增量同步版本、变更条数与耗时
  - `TrackDetailViewModel` / `AlbumDetailViewModel`：详情装载与封面解析
  - `InsightLiveActivity`：灵动岛状态推进

### 2. 日志查看方法

#### 方法 A：macOS 控制台 App (GUI，最推荐)
1. 启动控制台：`open -a Console`
2. 选择当前设备，点击顶部 **“开始流式传输”**；
3. 搜索栏输入过滤：`subsystem:com.vincentchyu.soniclens-bridge` 或进程 `SoniclensBridgeMac`。

#### 方法 B：终端命令行实时流式传输 (CLI)
```bash
# 查看所有 Debug/Info/Error 日志
log stream --predicate 'subsystem == "com.vincentchyu.soniclens-bridge"' --level debug

# 仅查看 API 网络请求与响应
log stream --predicate 'subsystem == "com.vincentchyu.soniclens-bridge" and category == "APIClient"' --level debug

# 仅查看 WebSocket 实时消息
log stream --predicate 'subsystem == "com.vincentchyu.soniclens-bridge" and category == "WebSocket"' --level debug
```

#### 方法 C：历史日志查看与导出
```bash
# 查看最近 1 小时的日志
log show --predicate 'subsystem == "com.vincentchyu.soniclens-bridge"' --last 1h

# 导出为日志文件
log show --predicate 'subsystem == "com.vincentchyu.soniclens-bridge"' --last 1h > ~/Desktop/soniclens_client.log
```

### 3. 本地数据与 SQLite 数据库路径

由于 macOS 端开发运行未强制开启 App Sandbox 隔离，本地资料库与缓存文件直接保存在用户主目录下的 `Application Support`：

- **macOS 本地资料库主目录**：
  `~/Library/Application Support/SonicLens/`
- **本地 SQLite 资料库索引文件**：
  `~/Library/Application Support/SonicLens/data/db/soniclens-library-index.sqlite`
- **快速在 Finder 中打开**：
  ```bash
  open ~/Library/Application\ Support/SonicLens/
  ```
- *注：若后续开启 App Sandbox 发布，沙盒路径将映射至 `~/Library/Containers/com.vincentchyu.soniclens-bridge.mac/Data/Library/Application Support/SonicLens/`。*

---

## 常见问题

- Bonjour 发现失败：
  - 确认客户端与服务端在同一局域网。
  - 检查服务端日志是否显示 Bonjour 已启动。
  - 检查 App 是否已授予“本地网络”权限。
- API 请求失败：
  - 确认连接页端口与服务端端口一致。
  - 检查 `/health` 是否返回 `{"status":"ok"}`。
  - 运行 `log stream --predicate 'subsystem == "com.vincentchyu.soniclens-bridge" and category == "APIClient"'` 查看请求明细与返回错误。
