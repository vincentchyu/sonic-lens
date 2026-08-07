# Dashboard V2 工作台骨架重构特性清单

- **日期**: 2026-03-26
- **范围**: `web/dashboard-v2` 的 `library`、`musicbrainz`、`pending-albums`
- **目标**: 把原先“宽留白 + 纵向下沉 + 非首屏闭环”的页面，统一收口成桌面端三段工作台：左侧队列、中央主画布、右侧检查器。

## 一、共享骨架

- 在 `src/app/styles/theme.css` 新增统一 workbench 规则：
  - 紧凑页头 `workbench-header`
  - 三列网格 `workbench-shell`
  - 左右 sticky 轨道 `workbench-sticky`
  - 独立滚动容器 `workbench-scroll`
  - 顶部锚定的 empty state / panel 结构
- 统一桌面密度，避免模块各自实现“看起来像三栏、实际依赖整页滚动”的伪工作台。

## 二、模块 A / Library

- 顶部大段 overview 与指标条收口成紧凑页头。
- 左列增加 `专辑 / 曲目` 队列切换，同一时刻只突出一个主队列，不再让两个列表并列抢注意力。
- 中列改成真正的 `Library Canvas`：
  - 专辑模式显示专辑结构、曲目行、差异与 `unlink`
  - 曲目模式显示曲目主键、位置、播放与收藏状态
- 右列改成固定 inspector：
  - `/api/library/sync` 增量同步入口
  - 当前对象摘要
  - 跨对象跳转动作（如从曲目定位专辑）

## 三、模块 D / MusicBrainz

- 移除首屏大 hero metrics，改成紧凑页头。
- 左列保留专辑队列，但收口为纯队列职责。
- 中列改成 `Candidate Stage`：
  - 当前专辑条
  - 候选列表
  - 当前候选详情同屏出现，不再上下断层
- 右列改成固定 inspector：
  - 当前状态
  - 本地 vs 候选差异摘要
  - 曲目对位 / JSON
  - 绑定版本与深度维护入口
- 消除“候选在上半屏、检查器掉到下半屏”的旧问题。

## 四、模块 C / Pending Albums

- 去掉 hero 与大指标区，改成紧凑页头。
- 左列改成 `Case Queue`：
  - `现场分组 / 工单队列` 分段切换
  - 只承担选案件职责
- 中列改成 `Case Canvas`：
  - `冻结证据 / 候选判断` 顶部切换
  - 候选列表与候选详情放回中列，不再塞进右侧
- 右列改成固定 inspector：
  - `context_stale`
  - 当前 MBID / resolved album / 完成时间
  - 刷新候选、刷新上下文、深度维护动作
  - 执行回执
- 保留 `context_stale` 只提示、不隐式刷新 的交互红线。

## 五、验证结果

- `cd web/dashboard-v2 && npm run lint` 通过
- `cd web/dashboard-v2 && npm run build` 通过
- 仍存在既有 Vite chunk size warning，但本次重构未引入新的构建错误

## 六、后续约束

- `library`、`musicbrainz`、`pending-albums` 后续新增能力时，默认继续落在“左队列 / 中画布 / 右检查器”骨架内。
- 桌面端首屏必须同时看到：当前选中对象、中列主内容、右侧状态与动作。
- 禁止再引入大 hero、垂直居中空态、或把检查器/主动作放到折叠以下。
- 工作台三列在桌面端必须共享同一可视高度；列内内容允许独立滚动，但外框高度不能参差。
- 顶层壳必须提供全屏操作模式，用于把三列一起扩展到整屏宽度，而不是只放大单个模块。
- 操作按钮、分页按钮、分段切换与状态徽标必须遵守统一高度体系，避免同屏出现尺寸凌乱的控件。
