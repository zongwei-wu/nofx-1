# Gateway MCP Server

NOFX Gateway MCP Server — 通过标准 MCP 协议直接交易。

**会话级认证**：每个用户通过 `gateway_auth` 提供自己的 API Key，多人共享同一进程互不干扰。

## 快速开始

```bash
cd gateway-mcp-server
npm install
npm run build
```

## 启动

```bash
# 本地
node dist/index.js

# 连接远程 NOFX
NOFX_API_URL="https://www.ttai.me" node dist/index.js
```

启动后 Agent 第一件事就是调用 `gateway_auth` 传入 API Key。

## Cursor 配置

```json
{
  "mcpServers": {
    "gateway": {
      "command": "node",
      "args": ["gateway-mcp-server/dist/index.js"],
      "env": {
        "NOFX_API_URL": "https://www.ttai.me"
      }
    }
  }
}
```

Agent 会自动发现 `gateway_auth` 工具，用户提供 Key 后即可交易。

## Hermes Web UI 配置

```yaml
gateway:
  command: node
  args:
    - gateway-mcp-server/dist/index.js
  env:
    NOFX_API_URL: "http://localhost:8080"
```

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `NOFX_API_URL` | NOFX 后端地址 | `http://localhost:8080` |

## 多用户隔离

```
用户 A → gateway_auth(api_key="nfx_sk_a1b2...") → 操作 A 的交易所
用户 B → gateway_auth(api_key="nfx_sk_c3d4...") → 操作 B 的交易所
```

每次会话独立认证，API Key 只存在内存中，不写入配置。

## 工具列表

| 工具 | 说明 | 
|------|------|
| `gateway_auth` | **必须先调用**：提供 API Key 建立会话 |
| `gateway_get_balance` | 查询交易所账户余额和净值 |
| `gateway_get_positions` | 查询当前持仓列表 |
| `gateway_get_market_price` | 查询交易对市价 |
| `gateway_get_klines` | 获取K线（仅 binance/okx） |
| `gateway_set_leverage` | 设置杠杆倍数 |
| `gateway_place_order` | 下单（需 confirmed: true） |
| `gateway_set_stop_loss` | 设置止损 |
| `gateway_set_take_profit` | 设置止盈 |
| `gateway_indicators_list` | 列出所有可用指标 |
| `gateway_indicators_compute` | 计算指定交易对指标 |
| `gateway_indicators_compare` | 多币种指标比较 |
| `gateway_strategy_list` | 列出我的策略 |
| `gateway_strategy_get` | 获取策略详情 |
| `gateway_strategy_create` | 创建策略 |
| `gateway_strategy_update` | 更新策略 |
| `gateway_strategy_delete` | 删除策略 |
| `gateway_strategy_validate` | 验证策略信号 |
| `gateway_strategy_activate` | 激活策略 |
| `gateway_backtest_run` | 启动回测 |
| `gateway_backtest_get` | 获取回测结果 |

## 安全

- **API Key 不写配置**：通过 gateway_auth 在会话中传入，内存存储，进程重启即清除
- **下单确认**：必须 `confirmed: true`
- **无配置写入**：不暴露交易所配置接口，只读 + 交易
- **用户隔离**：API Key 绑定用户，自动 scope 到该用户的交易所
