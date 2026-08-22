# GoScrapy Design Spec

审美方向：**深海声呐作战室（Sonar Ops）**。不是通用 SaaS 紫渐变，而是值班员在暗舱里看磷光回波的控制台。

## 调色

| Token | 值 | 用途 |
|---|---|---|
| `--bg-0` | `#071018` | 页面底 |
| `--bg-1` | `#0C1A24` | 侧栏 / 卡片 |
| `--bg-2` | `#122632` | 抬升面 |
| `--line` | `#1E3A47` | 分割 |
| `--cyan` | `#3EE0C5` | 主强调 / 在线 |
| `--amber` | `#F5B942` | 告警 / 进行中 |
| `--rose` | `#FF6B7A` | 失败 |
| `--text` | `#E7F4F2` | 主文字 |
| `--muted` | `#7FA3AE` | 次级文字 |

背景加极弱扫描线（2% 不透明度重复线性渐变）与右上角径向青晕，制造舱室进光。

## 字体

- Display：`Syne`（标题、大数字）
- Body：`IBM Plex Sans`
- Mono：`IBM Plex Mono`（选择器、URL、指标）

禁止 Inter / Roboto / Arial / Space Grotesk。

## 布局

全高应用壳：左侧 240px 导航（Logo「GOSCRAPY」+ 声呐环图标），主区 `w-full` 不限宽。768px 以下侧栏收为顶栏抽屉；480px 单列。

页面：

1. `/login` — 全屏居中玻璃卡片（唯一允许的居中限宽例外）
2. `/dashboard` — 集群大屏：节点卡 + ECharts 速率/失败率
3. `/rules` — 规则列表
4. `/rules/:id` — 可视化配置器：左截图热区，右字段与候选选择器
5. `/tasks` — 任务管理
6. `/tasks/:id` — 任务详情 + 结果表
7. `/results` — 全局结果
8. `/proxies` — 代理池

## 组件

- 节点卡：磷光描边，CPU/内存条用 cyan，失败率超 5% 切 amber/rose
- 按钮：实心 cyan / 幽灵描边 / 危险 rose；hover 外发光 8px
- 表格：深底斑马，表头大写 tracking
- Toast：Naive message，可关闭 + 5s 消失
- 危险操作：Naive Dialog 确认，禁止原生 alert
- 规则配置器：截图 `object-fit: none`，热区绝对定位，hover 描边 cyan，选中 amber

## 成本可见性

V1 无计费 API。LLM 开关默认关；若未来打开，触发按钮必须显示预估费用。
