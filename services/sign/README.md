# sub2api-sign

`sub2api-sign` 是面向 `sub2api` 的外挂式每日签到服务。它通过主站“自定义菜单页面”以 iframe 方式接入，不修改主站源码，也不直接写主站业务库发放余额。

## 架构

- `sub2api` 主站：负责登录态、菜单入口和 iframe 承载。
- `sub2api-sign-api`：负责签到状态、幂等、发奖、限流、补偿。
- `sub2api-sign-web`：负责嵌入式签到页面和动画。
- `sign_prod`：签到服务独立数据库，保存签到记录、奖励尝试和风控事件。
- Redis：保存短时 session、分布式锁和限流计数。

## 关键原则

- 前端不决定奖励金额。
- 签到服务不直接改 `users.balance`。
- 发奖优先走主站 `create-and-redeem` 管理接口。
- 每日签到使用 `UNIQUE(sub2api_user_id, sign_date)` 保证一次性。
- 奖励使用 `reward_code + Idempotency-Key` 做业务幂等。
- 失败发奖会落库并由补偿任务重试。

## 接口

- `GET /healthz`
- `GET /embed/bootstrap`
- `GET /api/sign/status`
- `POST /api/sign/checkin`
- `GET /api/sign/history`

## 配置

核心环境变量：

- `SIGN_DATABASE_URL`
- `SIGN_REDIS_URL`
- `SIGN_EMBED_TOKEN`
- `SUB2API_BASE_URL`
- `SUB2API_ADMIN_API_KEY`
- `SUB2API_GRANT_MODE`
- `SIGN_REWARD_RULE`
- `SIGN_BONUS_DAY3`
- `SIGN_BONUS_DAY7`
- `SIGN_BONUS_DAY15`
- `SIGN_BONUS_DAY30`
- `SIGN_RECOVERY_INTERVAL`
- `SIGN_RECOVERY_BATCH_SIZE`

默认奖励规则：

```text
0.50-1.50:45,1.51-3.50:30,3.51-6.00:17,6.01-9.00:7,9.01-10.00:1
```

## 数据表

- `sign_checkins`：核心签到记录。
- `sign_reward_attempts`：每次发奖请求和响应。
- `sign_risk_events`：高频或身份异常事件。
- `sign_user_stats`：用户连签与累计奖励缓存。

## 侧栏接入

建议菜单配置：

- 菜单名称：`每日签到`
- 可见角色：`普通用户`
- 页面 URL：`https://ai.gunddam.dpdns.org/external/sign/?k=<SIGN_EMBED_TOKEN>`

主站会继续向 iframe 追加 `user_id/token/theme/lang/ui_mode` 等上下文参数。签到服务会用 `token` 向主站反查用户，再建立自己的短时 session。

## 验证

后端：

```bash
go test ./...
```

前端：

```bash
cd web
npm ci
npm run build
```
