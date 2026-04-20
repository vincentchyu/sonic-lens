# SonicLens

SonicLens 是一个围绕“音乐轨迹属于你自己”构建的开源项目。它不只是一个 scrobble 工具，而是一整套完整产品：

- 核心后端：负责播放器监听、播放记录沉淀、AI 音眸解析、WebSocket 实时推送与数据同步。
- Web 后台：负责服务管理、统计看板、历史数据浏览与日常运维入口。
- 原生三端客户端：基于纯原生 SwiftUI 构建 macOS、iPadOS、iPhone 三端体验，用于连接家庭局域网中的 SonicLens 服务端，提供更完整的浏览、播放中、歌词、音眸与分享能力。

如果你想了解这个项目的精华，可以把它理解成一句话：

> 把你分散在不同播放器里的听歌历史，沉淀成你自己的本地音乐资产，并用 AI 重新组织成可浏览、可分析、可分享的个人音乐档案。

## 项目亮点

- 跨播放器沉淀音乐历史，统一接入 Apple Music、Audirvana、Roon 等播放来源。
- 本地优先保存听歌资产，核心数据由你自己掌控，不依赖某一个平台继续存在。
- AI 音眸解析能力，支持歌词、歌曲、专辑的深度分析与结构化洞察。
- 实时体验完整，后端变化可以通过 WebSocket 秒级推送到 Web 与客户端。
- 客户端不是壳应用，而是独立设计的原生三端产品，适合日常长期使用。

## 项目结构

这个仓库目前主要由三块组成：

### 1. 核心后端

位置：

- `main.go`
- `api/`
- `internal/`
- `core/`
- `config/`

主要职责：

- 监听播放器状态并进行 scrobble / 收藏状态同步
- 保存本地音乐资料与统计数据
- 提供 REST API、WebSocket、资料库同步接口
- 接入 AI 模型，生成歌曲与专辑 insight
- 为 Web 端与三端客户端提供统一数据服务

### 2. Web 后台 / 管理端

位置：

- `templates/`
- `static/`

主要职责：

- 展示当前播放、统计看板、历史数据
- 承担后台管理与日常维护入口
- 作为后端能力的可视化管理界面

### 3. 原生三端客户端 `soniclens-bridge`

位置：

- `soniclens-bridge/`

这是项目非常重要的一块。它不是简单的移动端适配，而是一套纯原生 SwiftUI 的局域网 Bridge 客户端，包含：

- macOS 客户端
- iPadOS 客户端
- iPhone 客户端

三端共享 `SoniclensCore` 网络层、模型层与状态管理，同时按端提供不同容器与交互方式。

## 三端客户端能做什么

`soniclens-bridge` 面向的是“使用者体验”，不是单纯开发调试工具。它目前承载的核心能力包括：

- 连接局域网中的 SonicLens 服务端，支持 Bonjour 自动发现与手动输入地址
- 浏览首页统计、趋势、热门内容与最近播放
- 浏览资料库，包括专辑、曲目、收藏、未上报等数据视图
- 查看当前播放、歌词、专辑信息与音眸解析
- 在 iPhone / iPad / macOS 上获得一致但不雷同的原生体验
- 支持分享海报、长图导出、Insight 结果消费等偏产品化的能力

如果你是第一次了解这个项目，这部分基本就是它和普通“音乐记录脚本”最大的区别。

## 效果展示

我们为所有端点提供了一致但深度适配的原生体验，包含 **Web 后台**、**Mac 桌面客户端** 以及 **iPhone 移动端**。

### 1. Web 后台管理

Web 后台主要用于大屏数据可视化展现、系统管理与日常运维入口。

<details>
<summary>点击展开查看 Web 端截图展示</summary>

| | |
| :---: | :---: |
| <img src="static/img/home2.png" width="100%"> | <img src="static/img/home1.png" width="100%"> |
| <img src="static/img/home3.png" width="100%"> | <img src="static/img/lens1.png" width="100%"> |
| <img src="static/img/lrc1.png" width="100%"> | <img src="static/img/lrc2.png" width="100%"> |

</details>

### 2. Mac 原生客户端

基于 SwiftUI 开发，深度融合 macOS 原生体验，支持毛玻璃视图体验和更完整的桌面多栏视图交互。

<details>
<summary>点击展开查看 Mac 端截图展示 (平铺布局)</summary>

| | |
| :---: | :---: |
| <img src="static/img/apple/MAC/1-home1.png" width="100%"> | <img src="static/img/apple/MAC/1-home2.png" width="100%"> |
| <img src="static/img/apple/MAC/1-home3.png" width="100%"> | <img src="static/img/apple/MAC/1-home4.png" width="100%"> |
| <img src="static/img/apple/MAC/2-专辑1.png" width="100%"> | <img src="static/img/apple/MAC/2-专辑2.png" width="100%"> |
| <img src="static/img/apple/MAC/2-专辑3.png" width="100%"> | <img src="static/img/apple/MAC/3-曲目1.png" width="100%"> |
| <img src="static/img/apple/MAC/3-曲目2.png" width="100%"> | <img src="static/img/apple/MAC/3-曲目3.png" width="100%"> |
| <img src="static/img/apple/MAC/4-音眸1.png" width="100%"> | <img src="static/img/apple/MAC/5-播放1.png" width="100%"> |
| <img src="static/img/apple/MAC/5-正在播放1.png" width="100%"> | <img src="static/img/apple/MAC/5-正在播放2.png" width="100%"> |
| <img src="static/img/apple/MAC/5-正在播放3.png" width="100%"> | <img src="static/img/apple/MAC/5-正在播放4.png" width="100%"> |
| <img src="static/img/apple/MAC/5-正在播放5.png" width="100%"> | |

</details>

### 3. iPhone 原生移动端

同样基于 SwiftUI 打造，为移动设备而生的阅读与正在播放体系，兼顾密集资料库与单曲歌词解析的沉浸展示。

<details>
<summary>点击展开查看 iPhone 常规界面展示 (三列并排)</summary>

| | | |
| :---: | :---: | :---: |
| <img src="static/img/apple/IPHONE/1-home1.PNG" width="100%"> | <img src="static/img/apple/IPHONE/1-home2.PNG" width="100%"> | <img src="static/img/apple/IPHONE/2-专辑1.PNG" width="100%"> |
| <img src="static/img/apple/IPHONE/3-曲目1.PNG" width="100%"> | <img src="static/img/apple/IPHONE/3-曲目3.PNG" width="100%"> | <img src="static/img/apple/IPHONE/正在播放1.PNG" width="100%"> |
| <img src="static/img/apple/IPHONE/正在播放2.PNG" width="100%"> | | |

</details>

<details>
<summary>点击展开查看 iPhone 超长沉浸解析图 (独立陈列)</summary>

针对长文本阅读（如长歌词、单曲音眸解读、专辑音眸洞察），可生成专属的高清晰度长图：

| | |
| :---: | :---: |
| <img src="static/img/apple/IPHONE/3-曲目4-腰-晚春-歌词.PNG" width="100%"> | <img src="static/img/apple/IPHONE/3-曲目5-腰-晚春-音眸.PNG" width="100%"> |
| <img src="static/img/apple/IPHONE/3-曲目2腰-晚春-信息.PNG" width="100%"> | <img src="static/img/apple/IPHONE/2-专辑2-腰-24'相见恨晚-信息.PNG" width="100%"> |
| <img src="static/img/apple/IPHONE/2-专辑3腰-24'相见恨晚-专辑音眸.PNG" width="100%"> | |

</details>

## 快速开始

### 1. 启动后端服务

先准备配置文件 `config/config.yaml`，至少补齐你自己的 Last.fm 信息：

```yaml
lastfm:
  apiKey: "YOUR_API_KEY"
  sharedSecret: "YOUR_SHARED_SECRET"
  userUsername: "YOUR_USERNAME"
  userPassword: "YOUR_PASSWORD"

scrobblers: ["Apple Music", "Audirvana", "Roon"]
```

可选依赖：

- `Redis`：用于缓存与部分状态加速
- `media-control`：部分播放器场景会用到

示例：

```bash
brew install redis
brew install media-control
```

运行方式：

```bash
# 方式一：前台调试
go build -o sonic-lens
./sonic-lens

# 方式二：使用仓库脚本部署为后台服务
sh shell/script/build_sonic-lens_launchctl.sh
sh shell/script/start_sonic-lens.sh
```

启动后，你可以使用 Web 端或客户端连接它。

### 2. 打开 Web 后台

后端起来后，可以直接访问项目的 Web 界面进行浏览与管理。具体端口以你的本地配置和启动日志为准。

Web 端适合：

- 先确认服务是否正常工作
- 查看统计看板和播放数据
- 做后台管理与运维操作

### 3. 构建三端客户端

客户端工程位于 `soniclens-bridge/`，目前包含三个 Scheme：

- `SoniclensBridgeMac`
- `SoniclensBridgePad`
- `SoniclensBridgePhone`

前置要求：

- macOS
- Xcode
- Xcode Command Line Tools
- [XcodeGen](https://github.com/yonaskolb/XcodeGen)
- 如需真机安装，需要可用的 Apple Developer Team

先生成 Xcode 工程：

```bash
cd soniclens-bridge
xcodegen generate
```

说明：

- `soniclens-bridge/SoniclensBridge.xcodeproj` 是生成产物
- 真正的工程定义在 `soniclens-bridge/project.yml`
- 如果你改了 target、scheme、Info.plist 生成配置，应该先改 `project.yml` 再重新生成工程

## 三端打包指引

### macOS

Xcode 中选择：

- Scheme：`SoniclensBridgeMac`
- Destination：`My Mac`

也可以命令行构建：

```bash
xcodebuild \
  -project soniclens-bridge/SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgeMac \
  -configuration Debug \
  -sdk macosx \
  build \
  CODE_SIGNING_ALLOWED=NO
```

### iPadOS

Xcode 中选择：

- Scheme：`SoniclensBridgePad`
- Destination：对应 iPad 真机或 iPad Simulator

命令行构建示例：

```bash
xcodebuild \
  -project soniclens-bridge/SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePad \
  -destination 'generic/platform=iOS' \
  build
```

### iPhone

Xcode 中选择：

- Scheme：`SoniclensBridgePhone`
- Destination：对应 iPhone 真机或 iPhone Simulator

命令行构建示例：

```bash
xcodebuild \
  -project soniclens-bridge/SoniclensBridge.xcodeproj \
  -scheme SoniclensBridgePhone \
  -configuration Debug \
  -destination 'generic/platform=iOS' \
  build \
  CODE_SIGNING_ALLOWED=NO
```

## 推荐阅读

如果你想继续深入：

- 客户端打包与启动说明：`soniclens-bridge/Docs/PACKAGING_AND_LAUNCH.md`
- 客户端构建与验证：`soniclens-bridge/Docs/BUILD_AND_VERIFY.md`
- 客户端架构说明：`soniclens-bridge/Docs/ARCHITECTURE.md`
- 客户端模块边界：`soniclens-bridge/Docs/CLIENT_MODULE_BOUNDARY.md`
- API 映射：`soniclens-bridge/Docs/API_MAPPING.md`

## 适合谁

这个项目会比较适合下面几类人：

- 想把自己的听歌历史长期保存在本地的人
- 同时使用多个播放器，希望统一沉淀与分析的人
- 对音乐资料、歌词、专辑文本分析、个人音乐资产感兴趣的人
- 想参考一个同时包含 Go 后端、Web 管理端、SwiftUI 三端客户端的完整开源项目的人

## 当前定位

SonicLens 不是单点工具，而是一套完整音乐系统：

- 后端负责采集、整理、同步、分析
- Web 负责管理、查看、运维
- 三端客户端负责真正面向用户的日常体验

这也是这个仓库最值得被开源读者一眼看到的地方。

## SigNoz 日志采集

当前应用侧保持 `zap + lumberjack` 本地文件日志策略，不在应用内直连 OTLP logs exporter。

如果需要把日志上报到本地 SigNoz，推荐使用 OpenTelemetry Collector 的 `filelog` receiver 采集 `./.logs/go_lastfm-scrobbler.log*`，再转发到 SigNoz OTLP `4317`。

示例配置：

- `config/otelcol/filelog_signoz.yaml.example`
