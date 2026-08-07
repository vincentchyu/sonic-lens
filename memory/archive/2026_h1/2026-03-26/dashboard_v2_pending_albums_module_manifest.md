# Dashboard V2 Pending Albums 模块特性清单

- 日期：2026-03-26
- 范围：`web/dashboard-v2/src/features/pending-albums/*`
- 目标：完成前端重构第 6 阶段 `模块 C / pending-albums`，把旧模板中的待归因专辑流程重构为独立的 v2 归因工作台。

## 本次落地

- 新增 `pending-albums` feature 的完整分层：
  - `api-client/pendingAlbumsApi.ts`
  - `domain/pendingAlbums.types.ts`
  - `domain/pendingAlbums.transformers.ts`
  - `state/usePendingAlbums.ts`
  - `views/PendingAlbumsPage.tsx`
  - `views/pending-albums.css`
- 页面结构不再沿用 v1 的弹窗链路，改为“现场入口 + 冻结证据 + 候选执行”的三栏工作台：
  - 左栏：待判分组与工单队列并排收口
  - 中栏：冻结证据、实时差异、播放流水、收藏事件
  - 右栏：MusicBrainz 候选判断、版本绑定、深度维护与执行回执
- `pending-albums` 保持与后端一致的“显式冻结”模型：
  - 先从 `/api/pending-albums` 读取现场分组
  - 通过 `POST /api/pending-albums/work-items` 将现场冻结为工单
  - 工单详情使用 `/api/pending-albums/work-items/:id` 读取冻结上下文
  - 现场变化只通过 `context_stale` 暴露，不做隐式刷新
  - 冻结重置必须显式调用 `POST /api/pending-albums/work-items/:id/refresh-context`
- 候选区不再自动偷偷搜索 MusicBrainz，而是由用户显式触发“刷新候选”，保持工作台节奏可控。
- 前端新增候选命中分与覆盖摘要：
  - 标题覆盖率
  - 轨序覆盖率
  - 未命中的冻结曲名列表
- 深度维护执行后，右侧固定回显：
  - `resolved_album_id`
  - 复用已听曲目数
  - 新建曲目数
  - `track_album` 写入数
  - 播放回放量与收藏回填量
- 所有动作接入 React Query mutation，并在成功后统一失效：
  - `pending-albums` 自身查询
  - `library` 查询
  - `musicbrainz` 查询

## 设计判断

- 这个模块的重点不是“搜到哪个候选”，而是“先让用户分清冻结证据和实时现场”。
- 因此页面先展示工单的证据链、`context_stale` 与刷新决策，再进入候选判断与危险动作。
- 候选搜索被改成显式动作，是为了避免切工单时自动触发远端检索，破坏节奏并制造误导。
- 中栏把证据拆成三种视图：
  - 证据矩阵：看哪些曲名和来源最有代表性
  - 播放流水：看现场到底发生了什么
  - 收藏事件：看哪些用户动作需要后续补写

## 关键事实来源

- 页面入口：`web/dashboard-v2/src/features/pending-albums/views/PendingAlbumsPage.tsx`
- 数据解析：`web/dashboard-v2/src/features/pending-albums/domain/pendingAlbums.transformers.ts`
- 查询与动作：`web/dashboard-v2/src/features/pending-albums/state/usePendingAlbums.ts`
- 接口基线：`api/api.md`
