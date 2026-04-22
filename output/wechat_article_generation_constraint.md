# WeChat 文章生成约束

这份文件定义 SonicLens 公众号文章的生成、引用和归档规则。  
目标是保证每次生成都能落到独立文章目录中，同时保留 `output/` 作为稳定兼容入口。

## 目录规范

- 当天生成的公众号文章必须放入独立目录，命名格式为 `sonic-lens_article_YYYYMMDD`。
- 例如 2026 年 4 月 21 日生成的文章目录应为 `sonic-lens_article_20260421`。
- 文章本体资源不得再直接散落到 `output/` 根目录。
- `output/` 只允许保留兼容入口、约束文件和指向文章目录的符号链接。

## 文章资源归属

文章目录内应至少包含以下内容：

- `wechat_article.md`：公众号正文的 Markdown 源文件。
- `wechat_article.html`：公众号可发布的最终 HTML。
- `wechat_article_body*.html`：正文片段，用于组装发布 HTML，允许按迭代保留版本后缀。
- `wechat_article_preview*.html`：带预览壳的本地审核页，允许按迭代保留版本后缀。
- `wechat_article_reader*.html`：读者最终视角版，用于确认最终阅读效果，允许按迭代保留版本后缀。
- `wechat_preview*.html`：本地预览入口页，允许保留兼容页和版本后缀。
- `wechat_assets/`：公众号文章专用素材，例如分享图、长图图块等。
- `wechat_preview_assets*/`：合图后的预览素材，允许按迭代保留版本后缀。

## 引用路径规则

### 1. Markdown 源文件

- `wechat_article.md` 中，引用仓库公共静态资源时使用相对路径 `../static/...`。
- `wechat_article.md` 中，引用文章目录内部资源时使用 `./wechat_assets/...`。
- 禁止把文章正文直接写成 `output/...` 绝对依赖。

### 2. 公众号 HTML

- `wechat_article.html`、`wechat_article_body*.html`、`wechat_article_preview*.html`、`wechat_article_reader*.html` 中，文章目录内部素材统一使用 `wechat_assets/...` 或 `wechat_preview_assets.../...`。
- 不得继续引用旧的 `output/wechat_assets/...` 前缀。
- 若 HTML 需要回退到仓库公共静态资源，应使用可相对解析的 `static/...` 或同层目录引用，避免写死上层输出目录。

### 3. 预览与兼容入口

- `output/wechat_article.md`、`output/wechat_article.html`、`output/wechat_preview*.html` 等文件应当是指向文章目录的符号链接。
- `output/wechat_assets` 与 `output/wechat_preview_assets*` 也应仅作为符号链接存在。
- 预览文件中显示的资源路径必须与文章目录内的真实文件一致。

## 生成流程约束

1. 先在文章目录中生成 Markdown、正文 HTML、预览 HTML 和读者版 HTML。
2. 再把合图素材放入文章目录的 `wechat_assets/` 与 `wechat_preview_assets_v2/`。
3. 最后在 `output/` 中更新兼容符号链接，保持原有打开方式可用。
4. 公众号草稿同步必须在用户确认最终读者版之后再执行。

## 发布前检查

- 检查文章目录是否包含当日命名的完整资源集合。
- 检查 `wechat_article.md` 的图片引用是否全部可解析。
- 检查 HTML 内是否存在 `output/wechat_assets/` 这类旧路径。
- 检查 `output/` 中对应入口是否仍然指向最新文章目录。
- 检查最终读者版的内容是否只保留给公众号读者可见的信息，不包含内部制作说明。

## 约束原则

- 公众号文章以“读者最终看到的内容”为准，不以内部制作过程为正文。
- 图片宁可少而整洁，也不要为了数量破坏排版。
- 复杂长图优先做总览板，必要时拆分为独立分享图区。
- 只有在用户确认最终读者版后，才同步公众号草稿箱。
