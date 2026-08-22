# API Contract Gate

> 时间：2026-08-22 (GMT+8)
> 判定：本项目 V1 无按量计费外部 API。

| Provider | 用途 | 验证状态 | 说明 |
|---|---|---|---|
| PostgreSQL 16 | 规则/任务/结果 | VERIFIED（本地容器） | 官方镜像，compose 健康检查 |
| Redis 7 | 队列/布隆/租约/选主 | VERIFIED（本地容器） | 官方镜像，compose 健康检查 |
| chromedp / headless-shell | 规则配置器快照 | UNVERIFIED until compose up | 无 API Key；失败时 Master 回退到 HTML 静态快照（仅 Mock 站），真实路径仍接通 |
| 代理池真实供应商 | 出口 IP | UNVERIFIED | `PROXY_POOL_MODE=mock` 默认；真实路径：`real` + `PROXY_LIST` |
| 公网站点抓取 | 任意 URL | UNVERIFIED | 默认抓 Mock 站；任务表单填公网 URL 即走真实 Fetcher |
| LLM | V2 抽取 | 关闭 | `LLM_ENABLED=false`，不实现调用 |

结论：Phase 3 先实现 Mock Provider + 真实 Fetcher/Renderer 接线；不猜测付费 API 响应。
