# Dashboard V2 MusicBrainz 模块特性清单

- 日期：2026-03-25
- 范围：`web/dashboard-v2/src/features/musicbrainz/*`
- 目标：完成前端重构第 5 阶段 `模块 D / musicbrainz`，将旧模板中的 MusicBrainz 候选搜索、确认绑定、深度维护动作重构为独立的 v2 维护台。

## 本次落地

- 新增 `musicbrainz` feature 的完整分层：
  - `api-client/musicbrainzApi.ts`
  - `domain/musicbrainz.types.ts`
  - `domain/musicbrainz.transformers.ts`
  - `state/useMusicBrainz.ts`
  - `views/MusicBrainzPage.tsx`
  - `views/musicbrainz.css`
- 页面结构不再沿用 v1 的弹窗式操作，改为三栏工作面：
  - 左栏：专辑队列与搜索
  - 中栏：MusicBrainz 候选判断
  - 右栏：检查器、曲目对位、原始 JSON 与执行区
- 专辑选择基于 `/api/albums` 与 `/api/albums/:id`，候选与维护动作为 `/api/musicbrainz/*`。
- 候选 JSON 在前端解析为可读结构，默认展示：
  - 日期、地区、包装、状态
  - 厂牌摘要
  - 碟数与曲目数
  - 封面/封底可用性
  - 文字表示（language/script）
- 右侧检查器支持三种视图：
  - 判断摘要
  - 曲目对位
  - 原始 JSON
- 增加候选与本地专辑的差异摘要：
  - 本地曲目数 vs 候选曲目数
  - 本地碟数 vs 候选碟数
  - 顺位对比得到的疑似错位数量
- 所有动作接入 React Query mutation，并在成功后统一失效：
  - `musicbrainz` 自身查询
  - `library` 查询（因为绑定/维护会改专辑与曲目关联视图）

## 设计判断

- 这个模块的核心不是“触发接口”，而是“帮助用户在绑定前建立判断”。
- 因此页面优先呈现候选判断信息，而不是先暴露危险动作按钮。
- 深度维护被放到最右侧执行区，并且只有在已确认版本后才允许触发，避免错误维护。

## 关键事实来源

- 页面入口：`web/dashboard-v2/src/features/musicbrainz/views/MusicBrainzPage.tsx`
- 数据解析：`web/dashboard-v2/src/features/musicbrainz/domain/musicbrainz.transformers.ts`
- 查询与动作：`web/dashboard-v2/src/features/musicbrainz/state/useMusicBrainz.ts`
- 接口基线：`api/api.md`
