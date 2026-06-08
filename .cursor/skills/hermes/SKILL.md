---
name: hermes
description: >-
  NOFX Hermes 自主交易助手：独立于 Web「AI 交易员」，通过 MCP 配置交易所与 AI 模型，
  启动后台 Runner 自主决策下单；对话内手动单笔仍用 hermes_trade 并需 confirmed。
  在用户提及 Hermes、自主交易、交易所 API、持仓、K 线、下单时使用。
---

# Hermes 交易 Skill

## 架构边界（硬性规则）

**禁止**调用以下 AI 交易员 API（与 Hermes 完全独立）：

- `/api/traders/*`（创建/启停/配置 AI 交易员）
- `/api/decisions?trader_id=...`、 `/api/status`、 `/api/account` 等 AI 交易员运行时接口
- Web「AI 交易员」的 start/stop 流程

**仅使用** Hermes 专用接口与 MCP 工具：

- 配置：`/api/exchanges`、`/api/models`（Hermes 权限组）
- 执行：`/api/hermes/*`
- 自主循环：`hermes_start_runner` / `hermes_stop_runner`

策略由 **Hermes Runner + `prompts/hermes.txt`** 决定，不由 Cursor 对话直接写策略文件。

## 安全约束（必须遵守）

1. **禁止**在回复中复述完整 API Key、Secret、Passphrase、私钥
2. 凭证**仅**通过 `hermes_save_exchange_credentials` 提交
3. **手动单笔**下单前必须向用户确认，并设置 `confirmed: true`
4. **自主模式**需用户明确授权：调用 `hermes_enable_autonomous` 且 `enabled: true`
5. 私钥类凭证（Hyperliquid / Aster）建议分两段收集

## 标准工作流

### 1. 认证

```
hermes_auth_status → 未登录则 hermes_login → 若 requires_otp 则 hermes_verify_otp
```

### 2. 门禁与配置

```
hermes_require_exchange_setup
→ 缺项则 hermes_get_exchange_card → hermes_test_exchange → hermes_save_exchange_credentials
→ hermes_list_models → hermes_update_settings（exchange_id、ai_model_id、扫描间隔等）
```

### 3. 开启自主交易（半自动）

用户**明确同意**后：

```
hermes_enable_autonomous(enabled: true)
→ hermes_start_runner
→ hermes_get_runner_status / hermes_get_decisions 监控
```

停止：`hermes_stop_runner`

Runner 内部下单**无需** `confirmed`；对话内 `hermes_trade` **仍须** `confirmed: true`。

### 4. 手动查询与单笔交易

| 工具 | 用途 |
|------|------|
| `hermes_get_klines` | K 线 |
| `hermes_get_balance` | 余额 |
| `hermes_get_positions` | 持仓 |
| `hermes_get_market_price` | 市价 |
| `hermes_trade` | 手动开/平/加/减仓（需 confirmed） |

## 支持的交易所

| ID | 交易 | K 线 |
|----|------|------|
| binance | ✅ | ✅ |
| okx | ✅ | ✅ |
| hyperliquid | ✅ | 暂不支持 |
| aster | ✅ | 暂不支持 |

卡片模板见 [exchange-cards.md](exchange-cards.md)。
