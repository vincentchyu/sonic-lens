# Admin 后台入口轻量拆分特性清单

## 背景

原 `templates/dashboard.html` 已增长到一万三千行以上，HTML、CSS 与 Vanilla JS 混在同一个模板文件中，后续继续迭代会增加回归风险。页面实际已承载统计、资料库、音眸、任务和维护等后台子页面，因此入口命名统一为 Admin。本次拆分遵循“零功能回归优先”：不引入构建工具，不改 API、DOM id、全局函数名或用户交互，只做机械搬迁、加载方式适配和入口命名收口。

## 变更摘要

- `templates/admin.html` 现在只保留页面 shell，通过 Go template partial 组合页面区域。
- Admin 局部模板已迁移到 `templates/admin/*.html`，包括 head、悬浮播放/歌词、侧栏、主内容区、弹窗、分享容器和脚本引用。
- Admin 内联 CSS 已迁移到 `static/admin/admin.css`。
- Admin 业务脚本已按原始顺序拆到 `static/admin/*.js`，继续使用普通全局脚本语义，保留现有 inline handler 与跨文件全局函数调用。
- `/static/admin/*` 由 Gin 目录静态路由提供，原有显式静态文件路由继续保留。
- 首页模板解析从单文件 `ParseFiles` 调整为主模板 + `templates/admin/*.html` partial 集合。
- `/` 继续作为历史首页入口，新增 `/admin` 作为后台总入口的语义化别名，两者共用同一个 Go template 渲染函数。
- `/api/dashboard/*` 继续表示仪表盘统计 API，不因后台入口命名改为 Admin 而改路径。
- 路由直连的独立页面归入 `templates/pages/`；`templates/recommendations.html` 与 `templates/report.html` 无后端引用，已删除。
- 新增 `static/admin/ui-state.js`，为列表页提供统一的 `renderAdminLoading`、`renderAdminEmpty`、`renderAdminError`，避免无数据时继续显示 loading 动效。

## 长期维护约束

- Admin 后续新增 UI 区块优先落到 `templates/admin/` 对应 partial，不要把大段 HTML 塞回 `templates/admin.html`。
- Admin 后续新增样式优先落到 `static/admin/admin.css`，除非是临时动态行内样式或第三方组件限制。
- Admin 后续新增脚本优先落到 `static/admin/` 的既有功能文件；若新增文件，必须同步维护 `templates/admin/scripts.html` 和静态资源可访问性。
- Admin 列表类页面的加载、空数据和错误状态优先使用 `ui-state.js` 统一渲染；详情弹窗或流式分析等特殊流程可以保留局部状态渲染，但不得把“空数据”伪装成 `.loading`。
- `dashboard` 命名只保留给具体仪表盘子页面、DOM section id 和 `/api/dashboard/*` 统计接口，避免迁移过程中破坏已有交互契约。
- 第一阶段仍保留全局函数和变量，后续如需命名空间化或事件绑定重构，必须作为单独阶段验证，不能和机械拆分混做。

## 验证记录

- `node --check static/admin/*.js` 通过。
- `go test ./api` 通过。
- `rg` 检查列表页中“暂无/没有”等空状态不再挂在 `.loading` 上。
- Go template 主模板 + partial 直接解析并执行成功。
- 临时本地服务验证 `/static/admin/*` 全部返回 `200`。
- in-app browser 加载拆分后的 Admin 入口成功，无 `ReferenceError` / `TypeError` 初始化错误；临时服务缺少真实 API，因此图表数据请求失败属于预期。
