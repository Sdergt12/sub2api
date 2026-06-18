# sub2api-risk

`sub2api-risk` 是面向 `sub2api` 的外挂式风控与风险分析子系统。

## 设计目标

- 不侵入 `weishaw/sub2api` 主镜像内部逻辑
- 独立读取 PostgreSQL / Redis / 日志进行分析
- 产出用户风险快照、风险事件、规则命中明细
- 提供风险看板 API 与后续管理界面
- 支持后续扩展为人工处置、白名单、自动告警与联动封禁

## 目录规划

```text
sub2api-risk/
├── README.md
├── .env.example
├── docker-compose.risk.yml
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── server/
│   │   └── router.go
│   ├── handler/
│   │   ├── dashboard_handler.go
│   │   ├── risk_user_handler.go
│   │   └── risk_event_handler.go
│   ├── service/
│   │   ├── dashboard_service.go
│   │   ├── risk_score_service.go
│   │   └── risk_event_service.go
│   ├── analyzer/
│   │   ├── engine.go
│   │   └── rules/
│   │       ├── burst_requests_rule.go
│   │       ├── cost_spike_rule.go
│   │       └── unique_ip_rule.go
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── risk_user_repository.go
│   │   │   └── risk_event_repository.go
│   │   └── redis/
│   │       └── window_repository.go
│   ├── model/
│   │   ├── risk_user.go
│   │   └── risk_event.go
│   └── dto/
│       ├── dashboard_dto.go
│       ├── risk_user_dto.go
│       └── risk_event_dto.go
├── migrations/
│   └── 001_init.sql
└── web/
    ├── package.json
    ├── index.html
    └── src/
        ├── main.ts
        ├── pages/
        │   ├── RiskOverviewPage.tsx
        │   ├── RiskUsersPage.tsx
        │   └── RiskEventsPage.tsx
        └── components/
            ├── RiskScoreCard.tsx
            └── UserRiskTable.tsx
```

## 数据源

### PostgreSQL
复用 `sub2api-ha` 中的 `postgres` 服务，读取主业务数据并新增独立风控表：

- `risk_user_snapshot`
- `risk_events`
- `risk_rule_hits`

### Redis
复用 `sub2api-ha` 中的 `redis` 服务，保存滑动窗口计数、IP 聚合、短周期去重等中间状态。

### 日志
通过只读挂载方式读取：

- `sub2api` 应用日志
- Nginx 访问日志（如果后续有单独入口层）

## 接入方式

建议通过独立 compose 文件或在现有 `docker-compose.yml` 中新增服务：

- `sub2api-risk-api`
- 可选：`sub2api-risk-web`

并加入现有网络：

- `sub2api-network`

## 第一阶段范围

第一版先实现：

1. 风险 API 骨架
2. 3 张基础表
3. 3 条规则
   - burst requests
   - cost spike
   - unique ip
4. 3 个基础页面
   - 风险总览
   - 可疑用户榜
   - 风险事件流

## 当前实现进度

当前已完成：

- `.env.example` 已补齐
- 后端已接入 PostgreSQL / Redis 基础连接层
- 已实现 `risk_user_snapshot` 与 `risk_events` 的 repository / service
- 以下接口已改为真实数据库查询，而非静态 mock：
  - `GET /api/risk/dashboard/overview`
  - `GET /api/risk/users`
  - `GET /api/risk/events`
- 前端页面已改为请求真实 API，并支持：
  - 风险总览加载与刷新
  - 用户列表筛选与分页
  - 事件列表筛选与分页
  - loading / error / empty 状态展示
- 已新增 `web/vite.config.js`，开发态通过 Vite 代理 `/api` 与 `/healthz`
- 已修正 `web/package.json` 构建脚本，统一为 `vite build`
- `docker-compose.risk.yml` 已为 `sub2api-risk-web` 注入 `VITE_PROXY_TARGET`

当前已新增首批规则刷新能力：

- 已落地 3 条基础风险规则：
  - burst requests
  - cost spike
  - unique ip
- 已支持从 PostgreSQL 源表/视图聚合风险数据并刷新：
  - `risk_user_snapshot`
  - `risk_events`
  - `risk_rule_hits`
- 已提供手动刷新接口：
  - `POST /api/risk/admin/refresh`
- 已支持启动时可选自动刷新：
  - `RISK_REFRESH_ON_START=true`

当前尚未完成：

- 日志解析、规则去重窗口、自动处置等后续能力

## 开发与验证

### 环境变量

优先参考项目内的 `.env.example`，核心变量如下：

- `RISK_DATABASE_URL`
- `RISK_REDIS_URL`
- `RISK_LISTEN_ADDR`
- `RISK_LOG_PATH`
- `RISK_READ_ONLY_MODE`
- `RISK_ENABLE_REFRESH_API`
- `RISK_REFRESH_TOKEN`
- `RISK_REFRESH_ON_START`
- `RISK_SOURCE_TABLE`
- `RISK_THRESHOLD_BURST_REQUESTS_1H`
- `RISK_THRESHOLD_COST_SPIKE_RATIO`
- `RISK_THRESHOLD_COST_SPIKE_FLOOR_1H`
- `RISK_THRESHOLD_UNIQUE_IP_24H`

如果希望和 `sub2api-ha` 维持一致风格，也可以通过：

- `DATABASE_URL`
- `REDIS_URL`

再透传给：

- `RISK_DATABASE_URL`
- `RISK_REDIS_URL`

### 风险刷新数据源要求

`RISK_SOURCE_TABLE` 必须指向一个 PostgreSQL 表或视图，并至少暴露以下字段：

- `user_id`
- `api_key_id`
- `model_name`
- `client_ip`
- `cost`
- `occurred_at`

当前刷新逻辑会基于最近 24 小时数据做聚合，计算：

- `request_count_1h`
- `request_count_24h`
- `cost_1h`
- `cost_24h`
- `unique_ip_24h`
- top 3 models
- `window_end`

### 首批规则说明

当前规则引擎会对每个用户聚合结果执行以下规则：

1. `burst_requests`
   - 条件：`request_count_1h >= RISK_THRESHOLD_BURST_REQUESTS_1H`
   - 默认严重级别：`high`
   - 默认分值：40

2. `cost_spike`
   - 条件：
     - `cost_1h >= RISK_THRESHOLD_COST_SPIKE_FLOOR_1H`
     - `cost_1h >= (cost_24h / 24) * RISK_THRESHOLD_COST_SPIKE_RATIO`
   - 默认严重级别：`warning`
   - 默认分值：35

3. `unique_ip`
   - 条件：`unique_ip_24h >= RISK_THRESHOLD_UNIQUE_IP_24H`
   - 默认严重级别：`warning`
   - 默认分值：25

风险等级当前映射如下：

- `score >= 70` => `high`
- `score >= 35` => `warning`
- 其他 => `normal`

### 手动刷新接口

当 `RISK_ENABLE_REFRESH_API=true` 且配置了 `RISK_REFRESH_TOKEN` 后，可调用：

- `POST /api/risk/admin/refresh`

请求头：

- `X-Risk-Refresh-Token: <your-refresh-token>`

示例：

```bash
curl -X POST http://localhost:8091/api/risk/admin/refresh \
  -H 'X-Risk-Refresh-Token: CHANGE_ME_REFRESH_TOKEN'
```

成功后会返回类似结构：

```json
{
  "status": "ok",
  "stats": {
    "scanned_users": 12,
    "updated_users": 12,
    "created_events": 5,
    "created_rule_hits": 5
  }
}
```

### 启动时刷新

若设置：

- `RISK_REFRESH_ON_START=true`

服务启动后会在 HTTP 服务监听前执行一次风险快照刷新，并在日志中输出刷新统计。

### 后端编译校验

建议在项目目录执行：

```bash
cd /root/sub2api/sub2api-risk && go build ./...
```

### 前端开发与联调

当前前端实际入口为：

- `web/src/main.jsx`

页面会直接请求以下真实接口：

- `GET /api/risk/dashboard/overview`
- `GET /api/risk/users`
- `GET /api/risk/events`

默认推荐使用 same-origin / Vite proxy 方式联调：

- 当未设置 `VITE_API_BASE_URL` 时，前端将直接请求相对路径 `/api/...`
- `web/vite.config.js` 会把 `/api` 与 `/healthz` 代理到 `VITE_PROXY_TARGET`
- 默认代理目标为：
  - `http://sub2api-risk-api:8091`

这意味着：

- 本地开发时无需修改后端 CORS
- Docker Compose 联调时，web 容器可直接通过服务名访问 API 容器
- 页面头部会展示当前 API Base，便于确认是否处于代理模式

如需显式指定外部 API，也可以设置：

- `VITE_API_BASE_URL=http://your-risk-api-host:8091`

此时前端会直接请求该地址，而不是走 Vite 代理。

### Compose 启动注意事项

当前环境下，`golang:1.22` 镜像内虽然存在 `/usr/local/go/bin/go`，但 `PATH` 里未必包含该目录。
因此 `docker-compose.risk.yml` 中 API 服务已改为使用绝对路径：

- `/usr/local/go/bin/go mod download`
- `/usr/local/go/bin/go run ./cmd/server`

前端联调时，`sub2api-risk-web` 服务会注入：

- `VITE_PROXY_TARGET=http://sub2api-risk-api:8091`

这样 `npm run dev` 启动后的 Vite 开发服务器就会把 `/api` 和 `/healthz` 自动代理到 API 容器。

### 主后台接入建议

推荐将 `sub2api-risk` 作为独立外挂式页面部署，再通过主后台“自定义菜单页面”接入。

推荐方式：

1. 独立启动 `sub2api-risk-api`
2. 可选启动 `sub2api-risk-web`
3. 对外暴露独立访问地址，例如：
   - `http://your-host:5173`
4. 在主后台中新增“自定义菜单页面”，指向该独立 URL
5. 优先使用新标签页打开；如果后台支持且样式隔离可接受，也可用 iframe 嵌入

这种方式的优点：

- 不需要侵入 `weishaw/sub2api` 主镜像
- 风控子系统可独立迭代与回滚
- 前后端部署与权限策略更容易单独演进
- 后续扩展人工处置、告警、白名单时不会干扰主后台结构

### 网络命名注意事项

`docker-compose.risk.yml` 当前使用的外部网络名为：

- `sub2api-ha_sub2api-network`

如果你的实际 Compose project name 不同，需要同步修改该 external network 名称。

## 后续扩展

- 用户详情页
- 白名单
- 人工备注
- 告警通知
- 自动处置联动
