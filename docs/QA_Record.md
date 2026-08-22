# QA Record

## Round 1 · 2026-08-22 23:31 (GMT+8)

**模式**：Mock / 离线  
**Cost**：¥0  
**技能**：`skills/test/webapp-testing.md`、`everything-claude-code/skills/e2e-testing/SKILL.md`、`everything-claude-code/skills/verification-loop/SKILL.md`

### 环境

`docker compose up --build -d` 后全部服务可用。Master / frontend / postgres / redis / mock-target / renderer 健康；worker-1、worker-2 在线。

入口：前端 http://localhost:27331 ；API http://localhost:27332 。

### 结果

| 检查 | 结果 | 备注 |
|---|---|---|
| Docker Build | PASS | master / worker / frontend / mock-target 镜像构建成功 |
| Health Check | PASS | `GET /api/v1/health` code=0，时间戳为北京时间 |
| Auth | PASS | 无 token → 401；错密 → 40102；admin 登录成功 |
| Mock API Smoke | PASS | `tests/api_smoke.py` 全绿 |
| 任务抓取 | PASS | 启动后约 1s 抽出 12 条商品；depth=1 继续入队详情 |
| 选择器推导 | PASS | 点击 Aurora Headphones → `#p-1 .title` unique；list_rule `.product-card` hit_count=12 |
| 快照 | PASS | CDP 1440×900，78 个可见节点 |
| 集群节点 | PASS | worker-1 / worker-2 上报 CPU/内存/Pages/Min |
| 代理池 | PASS | mock 三节点轮询记账 hits 5/4/4，不走假代理出口 |
| Playwright E2E | PASS | 3/3：壳层、登录巡航、创建任务 |
| 后端单测 | PASS | bloom / queue / parser / selector / proxy / api 等 |
| 计费调用 | PASS | 无外网计费 API |

### 本轮修复（联调）

1. mock 代理不得作为真实 HTTP Proxy，否则抓取全失败。改为 Dial 仅在 `real` 生效。
2. 任务必须先入队再标 running；无进度不收尾，避免空队列被立刻 succeeded。
3. Worker 代理命中写入 Redis，Master 大屏才能看到计数。
4. E2E 选择器对齐 Naive UI Select。

### 结论

Round 1 **PASS**。进入审计。

## Round 2 · 2026-08-22 23:35 (GMT+8)

**Cost**：¥0

针对 Go 审查指出的启动竞态、暂停误 Ack、心跳覆盖指标、布隆先占位、时间格式做了修复并热更新 Master/Worker。

复测：启动任务仍抽出 12 条；节点 `last_seen` 为 `yyyy-MM-dd HH:mm:ss`；心跳后 CPU/内存未被写成 0。

Round 2 **PASS**。
