# GoScrapy 实施路线图

> **文档性质**：定义 WHEN。需求真相源仍是 `docs/Requirements.md`。
> **状态**：已批准，进入实施
> **时间**：2026-08-22 (GMT+8)

---

## 0. 阶段顺序决策

**选择：Logic-First（交换 Phase 2 与 Phase 3）。**

理由：可视化规则配置器是一块由服务端快照数据模型（截图 + DOM 树 + BoxModel）推导出来的画布，前端热区与选择器面板无法在契约未冻结前凭空搭建；先锁定后端契约与选择器算法，再画 UI。

执行顺序：Phase 1 架构 → Phase 3 逻辑 → Phase 2 UI → Phase 4 QA → Phase 5 审计。

---

## 1. 阶段边界（强制，规模 10k–40k）

### MVP — 骨架可用（本轮必须先绿）

| Task ID | 内容 | 对应需求 |
|---|---|---|
| T-M1 | Git 初始化、`.gitignore`、compose 骨架、随机端口 | M-8 |
| T-M2 | Master HTTP API：登录 / 规则 CRUD / 任务创建 / 结果分页 | M-1, M-5, M-6, V-7 |
| T-M3 | Redis 优先级队列 + 租约 Pull | M-2 |
| T-M4 | Redis 位图布隆过滤器（任务级命名空间） | M-3 |
| T-M5 | 解析引擎 XPath / CSS / Regex | M-4 |
| T-M6 | Worker 抓取循环 + 结果落库 | M-1, M-6 |
| T-M7 | 前端三页：规则 / 任务 / 结果 | M-7 |
| T-M8 | 统一 zap Logger + GMT+8 | M-9 |
| T-M9 | Mock 目标站 + 一键 compose | M-8 |

### V1 — 完整交付（本轮一并交付，不得拖到 V2）

| Task ID | 内容 | 对应需求 |
|---|---|---|
| T-V1 | 服务端 CDP 快照三元组 + 选择器推导 | V-1 |
| T-V2 | 列表规则泛化（结构相似度聚类） | V-2 |
| T-V3 | 集群大屏 + WebSocket 指标推送 | V-3 |
| T-V4 | 代理池轮询 / 健康检查 / 驱逐 | V-4 |
| T-V5 | 每域 QPS + 429/403 自适应退避 | V-5 |
| T-V6 | Master Redis 选主 + Worker 租约回收 | V-6 |
| T-V7 | JWT 认证与测试账号 | V-7 |
| T-V8 | 单测 + Playwright E2E（Mock，¥0） | V-8 |
| T-V9 | `docs/API.md` 与实现对齐 | V-9 |

### V2 — 明确不在本轮交付

LLM 辅助抽取、规则 diff/回滚、CSV/Excel 导出、分布式链路追踪、通用 JS 渲染抓取。

---

## 2. 目录结构（冻结）

```
backend/                 # Go 单仓，master / worker 双入口
frontend-admin/          # Vue 3 + Naive UI + Tailwind
mock-target/             # 自建抓取目标站
tests/                   # Playwright E2E
docs/
docker-compose.yml
```

开发期端口（随机，避开 1848x / 8081）：

| 服务 | 宿主端口 |
|---|---|
| frontend-admin | 27331 |
| master HTTP | 27332 |
| master gRPC | 27333 |
| postgres | 27334 |
| redis | 27335 |
| mock-target | 27336 |
| renderer CDP | 27337 |

---

## 3. 进度追踪

- [x] T-M1 仓库与 compose
- [x] T-M2 Master API
- [x] T-M3 队列租约
- [x] T-M4 布隆去重
- [x] T-M5 解析引擎
- [x] T-M6 Worker 循环
- [x] T-M7 前端三页
- [x] T-M8 Logger / 时区
- [x] T-M9 Mock 站
- [x] T-V1 快照与选择器
- [x] T-V2 列表泛化
- [x] T-V3 监控大屏
- [x] T-V4 代理池
- [x] T-V5 自适应限速
- [x] T-V6 高可用
- [x] T-V7 JWT
- [x] T-V8 测试
- [x] T-V9 API 文档
