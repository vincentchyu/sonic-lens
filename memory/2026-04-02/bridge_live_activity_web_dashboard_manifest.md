# 特性清单：Bridge 音眸异步任务闭环、Live Activity 与 Web Dashboard 深度增强

## 1. 核心变更概览

本次更新是一次重大的架构升级，涉及 `soniclens-bridge` 客户端的任务协调机制、iOS 灵动岛/锁定屏幕实时活动支持，以及 `Web Dashboard` 管理后台的全面功能对齐。

## 2. 关键特性细节

### 2.1 Bridge: 音眸异步任务协调器 (InsightAnalysisCoordinator)
- **任务路由收口**：所有曲目/专辑音眸分析请求统一走异步 Job 模式，详情页通过 `Coordinator` 持有的 `activeJob` 状态驱动。
- **状态对账机**：实现 `reconcileIfNeeded`，在 App 唤醒、深链回写或 WebSocket 断连后，通过 `GET /api/insight-jobs/:id` 自动同步后端真实进度。
- **深链支持**：支持 `soniclens://insight-job/<id>` 协议，实现从系统通知或第三方应用直接回流到分析详情页。
- **持久化状态**：任务状态存储于 `UserDefaults`，允许 App 重启后恢复任务上下文。

### 2.2 Bridge: iOS Live Activity 实时活动
- **进度追踪**：引入 `InsightLiveActivityManager`，在 iPhone 锁定屏幕和灵动岛实时展示分析阶段（Phase）、模型、提供商及目标元数据。
- **推送更新 (Server-Push)**：支持通过 `Push Token` 实现由后端直接驱动的界面更新。
- **本地封面桥接**：由于 ActivityWidget 无法读取网络图片，通过 `LiveActivityArtworkStore` 将 `ResolvedArtworkResource` 异步下载至本地，并在任务状态变更时补发本地封面标识符。
- **退出策略**：针对不同终态（完成/失败/取消）设置差异化的 `dismissalPolicy`（180秒/45秒）。

### 2.3 Web Dashboard: 待处理专辑 (Pending Album) 工作流全面闭环
- **双轨维护路径**：
    - **MusicBrainz 模式**：通过搜索、选定 MBID、运行深度维护实现归因。
    - **手动编辑模式**：提供完整的曲目列表编辑器，支持碟号、曲序、艺人、时长修正。
- **上下文陈旧检测**：实时对比工单冻结统计与 live 统计，当检测到播放流水更新时，主动提示用户刷新工单详情，避免归因漂移。
- **曲目草稿推导**：基于播放流水与收藏事件的标题集合，自动去重并推导维护草稿，支持按物理位置自动排序。

### 2.4 Web Dashboard: 艺人与头像管理
- **独立页签**：新增 `Artist` 列表页签，集成搜索、分页与来源过滤。
- **头像上传**：集成文件上传接口，允许直接在 web 端更新艺人视觉资产。

## 3. 开发规范沉淀

### 3.1 客户端任务规范
- **禁止裸 Job 调用**：在 iOS 端启动耗时任务必须通过 `Coordinator` 封装，并考虑 `Live Activity` 反馈。
- **封面优先级**：实时任务界面首选 `artworkDescriptor` 里的本地 ID，只有在本地失效时才回退网络下载。

### 3.2 路由与状态规范
- **深链优先**：所有异步任务的 UI 反馈应尽量提供深链入口。
- **对账频率**：协调器的对账（Reconcile）间隔应遵循 `staleInterval` (默认12s)，避免高频轮询后端接口。

### 3.3 Web 端 UI 规范
- **按钮样式收口**：管理类的按钮统一使用 `.time-filter` 类名，保持半透明玻璃质感。
- **状态组件**：状态展示应复用 `renderPendingAlbumWorkItemStatusBadge` 逻辑，确保与后端枚举值视觉一致。

---
**日期**: 2026-04-02
**关联版本**: Bridge Core v1.5, Web Dashboard v2.0
