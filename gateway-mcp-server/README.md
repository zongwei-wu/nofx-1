# Gateway MCP Server

NOFX Gateway MCP Server — 通过标准 MCP 协议直接交易，无需配置交易所，无需登录。

## 快速开始

```bash
cd gateway-mcp-server
npm install
npm run build
```

## 启动

```bash
NOFX_API_KEY="nfx_sk_xxx" NOFX_API_URL="http://localhost:8080" node dist/index.js
```

## Cursor 配置

```json
{
  "mcpServers": {
    "gateway": {
      "command": "node",
      "args": ["gateway-mcp-server/dist/index.js"],
      "env": {
        "NOFX_API_URL": "https://www.ttai.me",
        "NOFX_API_KEY": "nfx_sk_your_key_here"
      }
    }
  }
}
```

## Hermes Web UI 配置

```yaml
gateway:
  command: node
  args:
    - gateway-mcp-server/dist/index.js
  env:
    NOFX_API_URL: "http://localhost:8080"
    NOFX_API_KEY: "nfx_sk_xxx"
```

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `NOFX_API_KEY` | API Key，从 NOFX Web UI 的 /access-keys 生成 | 必填 |
| `NOFX_API_URL` | NOFX 后端地址 | `http://localhost:8080` |

## 工具列表

| 工具 | 说明 | 
|------|------|
| `gateway_get_balance` | 查询交易所账户余额和净值 |
| `gateway_get_positions` | 查询当前持仓列表 |
| `gateway_get_market_price` | 查询交易对市价 |
| `gateway_get_klines` | 获取K线（仅 binance/okx） |
| `gateway_set_leverage` | 设置杠杆倍数 |
| `gateway_place_order` | 下单（需 confirmed: true） |
| `gateway_set_stop_loss` | 设置止损 |
| `gateway_set_take_profit` | 设置止盈 |

## 安全

- API Key 通过环境变量注入，不经过 MCP tool 参数
- 下单必须 `confirmed: true`
- 不暴露交易所配置写入接口，只读 + 交易
- 每个 API Key 绑定唯一用户，自动隔离
