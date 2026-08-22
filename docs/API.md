# GoScrapy HTTP / gRPC / WebSocket API

基准地址（开发期）：`http://localhost:27332`
前端入口：`http://localhost:27331`
认证：除 `/api/v1/health` 与 `POST /api/v1/auth/login` 外，均需 `Authorization: Bearer <jwt>`。

统一响应：

```json
{"code":0,"message":"ok","data":{}}
```

错误：

```json
{"code":40101,"message":"未登录或令牌失效"}
```

时间字段一律 `yyyy-MM-dd HH:mm:ss`（GMT+8）。

---

## 错误码表

| code | HTTP | 含义 |
|---|---|---|
| 0 | 200/201/202 | 成功 |
| 40001 | 400 | 参数校验失败 |
| 40101 | 401 | 未认证 |
| 40102 | 401 | 用户名或密码错误 |
| 40401 | 404 | 资源不存在 |
| 40901 | 409 | 资源冲突（规则名重复等） |
| 50001 | 500 | 内部错误 |
| 50301 | 503 | 依赖不可用（渲染器/队列） |

---

## 1. 认证

### POST `/api/v1/auth/login`

请求：

```json
{"username":"admin","password":"Admin@12345"}
```

响应：

```json
{"code":0,"message":"ok","data":{"token":"<jwt>","expires_in":86400,"username":"admin"}}
```

---

## 2. 健康检查

### GET `/api/v1/health`

```json
{"code":0,"message":"ok","data":{"status":"ok","role":"master","time":"2026-08-22 23:10:00"}}
```

---

## 3. 规则

### GET `/api/v1/rules?page=1&page_size=20&keyword=`

### POST `/api/v1/rules`

```json
{
  "name": "mock-shop-list",
  "start_url": "http://mock-target/list.html",
  "item_selector": ".product-card",
  "link_selector": "a.product-link",
  "fields": [
    {"name":"title","kind":"css","expr":".title","attr":"text"},
    {"name":"price","kind":"css","expr":".price","attr":"text"},
    {"name":"sku","kind":"regex","expr":"SKU-(\\d+)","attr":"text"}
  ],
  "respect_robots": true,
  "qps": 2
}
```

`kind` ∈ `xpath` | `css` | `regex`。

### GET `/api/v1/rules/:id`

### PATCH `/api/v1/rules/:id`（版本号 +1）

### DELETE `/api/v1/rules/:id`

### POST `/api/v1/rules/:id/preview`

```json
{"html":"<html>...","url":"http://mock-target/list.html"}
```

---

## 4. 任务

### POST `/api/v1/tasks` → 202

```json
{"name":"demo-crawl","rule_id":1,"seed_urls":["http://mock-target/list.html"],"max_depth":2,"concurrency":4}
```

### GET `/api/v1/tasks?page=1&page_size=20&status=`

### GET `/api/v1/tasks/:id`

### POST `/api/v1/tasks/:id/start`

### POST `/api/v1/tasks/:id/pause`

### POST `/api/v1/tasks/:id/cancel`

状态机：`created → running → paused|succeeded|failed|cancelled`。`paused` 可再 `start`。

---

## 5. 结果

### GET `/api/v1/tasks/:id/results?page=1&page_size=20`

### GET `/api/v1/results?task_id=&page=1&page_size=20`

---

## 6. 快照与选择器（可视化配置器）

### POST `/api/v1/snapshots`

```json
{"url":"http://mock-target/list.html"}
```

响应 `data`：

```json
{
  "snapshot_id":"snap_xxx",
  "width":1440,
  "height":900,
  "image_url":"/api/v1/snapshots/snap_xxx/image",
  "nodes":[{"node_id":12,"tag":"h2","text":"Aurora Headphones","box":{"x":24,"y":180,"w":280,"h":32}}]
}
```

### GET `/api/v1/snapshots/:id/image` → `image/png`

### POST `/api/v1/snapshots/:id/selectors`

```json
{"node_id":12}
```

响应：

```json
{
  "candidates":[
    {"kind":"css","expr":"#p-1 .title","unique":true,"score":0.98},
    {"kind":"xpath","expr":"//*[@id='p-1']//h2","unique":true,"score":0.9}
  ],
  "list_rule":{"item_selector":".product-card","field_selector":".title","hit_count":12}
}
```

---

## 7. 集群与代理

### GET `/api/v1/cluster/nodes`

### GET `/api/v1/cluster/metrics`

### GET `/api/v1/proxies`

### GET `/api/v1/queue/stats`

---

## 8. WebSocket

`GET /api/v1/ws/metrics`（同样带 Bearer，可用 `?token=`）

服务端每 ≤3s 推送：

```json
{"type":"metrics","ts":"2026-08-22 23:10:03","nodes":[{"id":"worker-1","cpu":12.4,"memory_mb":186,"pages_per_min":64,"fail_rate":0.02}]}
```

---

## 9. gRPC 控制面

`ControlPlane.Connect` 双向流。Worker 上报心跳与指标；Master 下发限速、规则热更新、优雅停机。任务体不走 gRPC。
