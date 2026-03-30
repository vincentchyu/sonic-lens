# Bridge Query 百分号编码修复清单

## 背景

Bridge 客户端在请求 `/api/track-lyrics` 时，曲名 `8+2+8 II` 中的 `+` 未被稳定编码为 `%2B`。后端按 query/form 语义解析时将 `+` 视为空格，导致实际收到的曲名变成 `8 2 8 II`，从而命中错误歌词。

## 本次修复

1. 将 `soniclens-bridge/SoniclensCore/Networking/APIClient.swift` 的 GET URL 构造收口到统一 `buildURL` 逻辑。
2. Bridge 共享网络层不再直接依赖 `URLComponents.queryItems` 的默认行为，而是显式构造 `percentEncodedQuery`。
3. query component 统一执行百分号编码，至少保证 `+`、`&`、`=` 不会以原字符进入 URL。
4. `nil` query 参数不再编码成裸 key，避免出现 `trackNumber&discNumber` 这类不完整查询串。

## 客户端约束

1. Bridge 侧所有 GET query 参数必须通过共享网络层统一编码，禁止在 ViewModel、Store、View 或调用点手写 query string。
2. 曲名、艺人名、专辑名等音乐元数据一律按“原样值”传入 `URLQueryItem`，由 `APIClient` 负责最终百分号编码，禁止调用方预先替换 `+` 为空格或自行做不一致转义。
3. 只要 query value 中可能出现 `+`、空格、`&`、`=`、`?`、`/` 等字符，就必须以“后端按百分号解码后得到原始字符串”为验收标准，不能以浏览器地址栏显示效果作为正确性依据。

## 影响范围

- `/api/track-lyrics`
- `/api/track-insight`
- `/api/artwork/resolve`
- 以及所有复用 `SoniclensCore/Networking/APIClient.swift` 的 GET 请求

## 验证结论

修复后，`8+2+8 II` 在客户端生成的 query 中会被编码为 `8%2B2%2B8%20II`，后端可稳定还原成原始曲名 `8+2+8 II`。
