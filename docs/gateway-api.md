# NOFX 交易所网关 API

NOFX 为外部 **Hermes** 等自主交易服务提供交易所网关：复用用户在 NOFX 配置的 `exchanges` 凭证，代为查询行情/持仓并执行交易。

与 Web「AI 交易员」（`/api/traders/*`、`AutoTrader`）**完全独立**。

## 快速开始

### 1. 用户侧（NOFX Web）

1. 登录 NOFX，在「交易所配置」中保存 Binance/OKX 等 API 凭证
2. 进入「API Keys」页面，生成 `nfx_sk_...`（仅显示一次）
3. 将 Key 配置到自建 Hermes 服务

### 2. Hermes 服务侧

```bash
export NOFX_GATEWAY_URL=https://your-nofx-host
export NOFX_API_KEY=nfx_sk_xxx
```

```go
import "nofx/gateway/client"

c := client.New(os.Getenv("NOFX_GATEWAY_URL"), os.Getenv("NOFX_API_KEY"))

// 探活 + 能力清单
caps, _ := c.Capabilities(ctx)

// 门禁：检查交易所是否已配置
exchanges, _ := c.ListExchanges(ctx)

// Agent 循环
balance, _ := c.GetBalance(ctx, "binance")
positions, _ := c.GetPositions(ctx, "binance")
klines, _ := c.GetKlines(ctx, "binance", "BTCUSDT", "1h", 100)

// 下单（API Key 路径无需 confirmed）
c.Trade(ctx, client.TradeRequest{
    ExchangeID: "binance",
    Action:     "open_long",
    Symbol:     "BTCUSDT",
    Quantity:   0.01,
    Leverage:   5,
})
```

## 认证

所有 Gateway 接口使用 API Key：

```
Authorization: ApiKey nfx_sk_...
```

每个 Key 绑定唯一 `user_id`，自动隔离交易所配置与持仓。

## 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/capabilities` | 能力清单 |
| GET | `/api/gateway/exchanges` | 交易所配置（脱敏） |
| PUT | `/api/gateway/exchanges` | 更新凭证（RSA 加密，与 Web 相同） |
| POST | `/api/gateway/exchanges/test` | 测试连接 |
| GET | `/api/gateway/klines` | K 线（binance/okx） |
| GET | `/api/gateway/balance` | 余额 |
| GET | `/api/gateway/positions` | 持仓 |
| GET | `/api/gateway/market-price` | 市价 |
| POST | `/api/gateway/trade` | 开/平/加/减仓 |
| POST | `/api/gateway/leverage` | 设置杠杆 |
| POST | `/api/gateway/stop-loss` | 止损 |
| POST | `/api/gateway/take-profit` | 止盈 |

### 指标

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/indicators/list` | 列出所有可用指标 |
| POST | `/api/gateway/indicators/compute` | 批量计算指标 |
| POST | `/api/gateway/indicators/compare` | 多币种指标比较 |

### 策略管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/strategies` | 列出我的策略 |
| POST | `/api/gateway/strategies` | 创建策略 |
| GET | `/api/gateway/strategies/:id` | 策略详情 |
| PUT | `/api/gateway/strategies/:id` | 更新策略 |
| DELETE | `/api/gateway/strategies/:id` | 删除策略 |
| POST | `/api/gateway/strategies/:id/validate` | 验证当前信号 |
| POST | `/api/gateway/strategies/:id/activate` | 激活策略 |
| POST | `/api/gateway/strategies/:id/pause` | 暂停策略 |
| GET | `/api/gateway/strategies/:id/signals` | 信号历史 |
| POST | `/api/gateway/strategies/:id/backtest` | 启动回测 |

### 回测

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/backtests` | 回测历史列表 |
| GET | `/api/gateway/backtests/:id` | 回测结果详情 |

完整 Swagger 定义见 [`docs/swagger.json`](swagger.json)。

## 交易 action

`open_long` / `open_short` / `add_long` / `add_short` / `close_long` / `close_short` / `reduce_long` / `reduce_short`

API Key 调用 **无需** `confirmed` 字段。

## 多用户

Hermes 服务为每个用户维护独立的 `gateway/client.Client` 实例，各自使用不同的 `nfx_sk_...`。

## 数据库迁移（可选）

Runner 下线后，可手动清理旧表：

```sql
DROP TABLE IF EXISTS hermes_settings;
```

## 与 AI 交易员边界

| 项 | AI 交易员 | Gateway |
|----|-----------|---------|
| 入口 | `/api/traders/*` | `/api/gateway/*` |
| 认证 | JWT + `ai_trader` 权限 | API Key |
| 策略/循环 | NOFX `AutoTrader` | 自建 Hermes |
| 决策日志 | `decision_logs/{trader_id}/` | Hermes 自有存储 |
