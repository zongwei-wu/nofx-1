# Hermes MCP Server

NOFX Hermes 交易 MCP Server，供 Cursor Agent 通过标准 MCP 协议调用交易与交易所配置接口。

## 环境要求

- Node.js >= 18
- 运行中的 NOFX 后端

## 安装与构建

```bash
cd hermes/mcp-server
npm install
npm run build
```

## Cursor 配置

项目根目录 [`.cursor/mcp.json`](../../.cursor/mcp.json) 已预置：

```json
{
  "mcpServers": {
    "hermes": {
      "command": "node",
      "args": ["hermes/mcp-server/dist/index.js"],
      "env": {
        "NOFX_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

修改 `NOFX_API_URL` 指向你的 NOFX 实例。

## 环境变量

| 变量 | 说明 |
|------|------|
| `NOFX_API_URL` | NOFX 后端地址，默认 `http://localhost:8080` |
| `NOFX_JWT` | 可选，预置 JWT（跳过 hermes_login） |
| `NOFX_USER_ID` | 配合 `NOFX_JWT` 使用 |
| `NOFX_EMAIL` | 可选，显示用 |
| `NOFX_PROJECT_ROOT` | 可选，项目根路径（读取 exchange-cards.md） |

## 认证

1. `hermes_login`：邮箱 + 密码
2. 若返回 `requires_otp`：调用 `hermes_verify_otp` 输入 Google Authenticator 验证码
3. `hermes_auth_status`：检查登录状态

## 工具列表

| 工具 | 说明 |
|------|------|
| `hermes_login` | 登录 |
| `hermes_verify_otp` | 2FA 验证 |
| `hermes_auth_status` | 认证状态 |
| `hermes_get_server_ip` | 服务器 IP（白名单） |
| `hermes_get_exchange_card` | 交易所配置卡片 |
| `hermes_list_exchanges` | 交易所列表（脱敏） |
| `hermes_test_exchange` | 测试连接（加密） |
| `hermes_save_exchange_credentials` | 保存凭证（加密） |
| `hermes_get_klines` | K 线 |
| `hermes_get_balance` | 余额 |
| `hermes_get_positions` | 持仓 |
| `hermes_get_market_price` | 市价 |
| `hermes_trade` | 开/平/加/减仓 |
| `hermes_set_leverage` | 设置杠杆 |
| `hermes_set_stop_loss` | 止损 |
| `hermes_set_take_profit` | 止盈 |
| `hermes_get_settings` | 读取 Hermes 设置与 Runner 状态 |
| `hermes_update_settings` | 更新交易所、AI 模型、扫描间隔等 |
| `hermes_enable_autonomous` | 开启/关闭半自动授权 |
| `hermes_start_runner` | 启动后台自主交易 Runner |
| `hermes_stop_runner` | 停止 Runner |
| `hermes_get_runner_status` | Runner 运行状态 |
| `hermes_get_decisions` | Hermes 决策日志 |
| `hermes_list_models` | 可选 AI 模型列表 |
| `hermes_require_exchange_setup` | 配置门禁检查 |

## Hermes 自主交易

Hermes 与 Web「AI 交易员」**完全独立**：

1. `hermes_require_exchange_setup` → 配置交易所与 AI 模型
2. 用户授权后 `hermes_enable_autonomous(enabled: true)`
3. `hermes_start_runner` 启动后台循环（策略由 `prompts/hermes.txt` 决定）
4. Runner 内部下单无需 `confirmed`；对话内 `hermes_trade` 仍须 `confirmed: true`

**禁止**通过 MCP 调用 `/api/traders/*` 等 AI 交易员接口。

## 安全说明

- 敏感凭证经 RSA-OAEP + AES-GCM 加密后传输，与 Web 端 [`web/src/lib/crypto.ts`](../../web/src/lib/crypto.ts) 一致
- MCP Server 不在日志中输出完整密钥
- 手动 `hermes_trade` 需 `confirmed: true`，且 Agent 应向用户二次确认

## 测试

```bash
npm test
```
