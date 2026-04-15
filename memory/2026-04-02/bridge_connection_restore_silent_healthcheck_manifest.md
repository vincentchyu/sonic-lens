# Bridge 连接恢复与静默健康检查策略

## 背景

Bridge 过去在重启后会直接回到连接页，用户需要手动“重新连接”。这会打断已建立的局域网交互路径，也让用户误以为上次连接是一次性会话。

## 目标

- 记住上次成功连接，启动后先静默健康检查。
- 健康检查成功则直接进入 dashboard。
- 健康检查失败不自动断开，而是进入“待决策”状态，由用户决定退出当前连接或重新连接。
- 连接建立后持续做静默健康检查，前台切回时优先做一次即时检查。

## 实现要点

- `AppStore` 新增：
  - `connectionRecoveryState`
  - `bootstrapConnectionIfNeeded()`
  - `performForegroundConnectionHealthCheckIfNeeded()`
  - 周期性的静默健康检查 Task
  - `autoRestoreLastSuccessfulConnection` 持久化开关
- 根入口改成三段式：
  - 恢复中：展示静默恢复页
  - 失效待决策：展示决策页
  - 正常：展示连接页或 dashboard
- 连接失效后会暂停 WebSocket 监听，避免继续后台重连；用户重新连接成功后再恢复实时监听。

## 验证

- 关闭 App 后重开，健康检查通过时应直进 dashboard。
- 服务端离线时重开，应该看到待决策页，而不是直接断开。
- 连接建立后，后台静默检查失败会触发待决策页。
- 用户点“退出当前连接”后，自动恢复开关会关闭，下次启动不再自动重连。

