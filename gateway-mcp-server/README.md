# Gateway MCP Server

NOFX Gateway MCP Server — 通过标准 MCP 协议直接交易。

**会话级认证**：默认通过 `gateway_auth` 传入 API Key；也可在环境变量中预设 `NOFX_API_KEY`（适合个人单机使用）。

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

启动后若未设置 `NOFX_API_KEY`，Agent 需先调用 `gateway_auth` 传入 API Key。

## Cursor 配置

配置文件二选一（JSON 格式相同）：

| 位置 | 作用域 |
|------|--------|
| `~/.cursor/mcp.json` | 全局，所有项目可用 |
| `.cursor/mcp.json`（项目根目录） | 仅当前项目 |

修改后重启 Cursor，或在 **Settings → MCP** 中刷新服务。

### 方式一：运行时认证（推荐，多用户 / 共享进程）

不在配置中写 API Key，由 Agent 调用 `gateway_auth` 建立会话：

```json
{
  "mcpServers": {
    "gateway": {
      "command": "node",
      "args": ["/绝对路径/nofx-1/gateway-mcp-server/dist/index.js"],
      "env": {
        "NOFX_API_URL": "https://www.ttai.me"
      }
    }
  }
}
```

`args` 请使用**绝对路径**，避免 Cursor 工作目录变化导致找不到 `dist/index.js`。

### 方式二：环境变量预设 Key（个人单机）

适合只有自己使用、不想每次对话都调 `gateway_auth` 的场景：

```json
{
  "mcpServers": {
    "gateway": {
      "command": "node",
      "args": ["/绝对路径/nofx-1/gateway-mcp-server/dist/index.js"],
      "env": {
        "NOFX_API_URL": "https://www.ttai.me",
        "NOFX_API_KEY": "nfx_sk_..."
      }
    }
  }
}
```

> 注意：`NOFX_API_KEY` 写在本地配置文件中，请勿提交到 Git。多人共用同一 MCP 进程时请用方式一。

## Hermes Web UI 配置

与 Cursor 相同，也支持 `gateway_auth` 或 `NOFX_API_KEY` 两种认证方式：

```yaml
gateway:
  command: node
  args:
    - /绝对路径/nofx-1/gateway-mcp-server/dist/index.js
  env:
    NOFX_API_URL: "http://localhost:8080"
    # NOFX_API_KEY: "nfx_sk_..."   # 可选，不设则运行时 gateway_auth
```

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `NOFX_API_URL` | NOFX 后端地址 | `http://localhost:8080` |
| `NOFX_API_KEY` | API Key（可选） | 无；未设置时需 `gateway_auth` |

## 认证方式对比

| 方式 | 适用场景 | 说明 |
|------|----------|------|
| `gateway_auth` | 多用户、共享 MCP 进程 | Key 仅存内存，进程重启即清除 |
| `NOFX_API_KEY` | 个人单机 | 写在 `env` 中，启动即认证，无需先调 `gateway_auth` |

## 多用户隔离

```
用户 A → gateway_auth(api_key="nfx_sk_a1b2...") → 操作 A 的交易所
用户 B → gateway_auth(api_key="nfx_sk_c3d4...") → 操作 B 的交易所
```

每次会话独立认证，API Key 只存在内存中，不写入配置。

## 工具列表

| 工具 | 说明 | 
|------|------|
| `gateway_auth` | 提供 API Key 建立会话（未设 `NOFX_API_KEY` 时必须先调用） |
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

- **API Key 优先不写配置**：推荐通过 `gateway_auth` 在会话中传入，内存存储，进程重启即清除
- **环境变量 Key**：`NOFX_API_KEY` 仅适合个人本地，勿提交到版本库
- **下单确认**：必须 `confirmed: true`
- **无配置写入**：不暴露交易所配置接口，只读 + 交易
- **用户隔离**：`gateway_auth` 模式下每次会话独立，API Key 自动 scope 到对应用户
