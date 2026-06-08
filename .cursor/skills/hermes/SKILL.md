---
name: hermes
description: >-
  NOFX Hermes 交易助手：通过 MCP 工具完成交易所配置、K 线查询、持仓查看、开平仓/加减仓。
  在用户提及交易、Hermes、交易所 API、持仓、K 线、下单时使用。
---

# Hermes 交易 Skill

## 安全约束（必须遵守）

1. **禁止**在回复中复述完整 API Key、Secret、Passphrase、私钥
2. 凭证**仅**通过 `hermes_save_exchange_credentials` 提交，不得写入代码或 Skill 文件
3. 下单前**必须**向用户确认：交易所、币种、方向、数量、杠杆，并获用户明确同意后设置 `confirmed: true`
4. 私钥类凭证（Hyperliquid / Aster）建议分两段收集，对齐卡片说明

## 标准工作流

### 1. 认证

```
hermes_auth_status → 未登录则 hermes_login → 若 requires_otp 则 hermes_verify_otp
```

也可通过环境变量 `NOFX_JWT` + `NOFX_USER_ID` 预置会话。

### 2. 交易所配置

```
hermes_list_exchanges → 未配置则 hermes_get_exchange_card(exchange_id)
→ 按卡片逐步引导用户 → hermes_get_server_ip（白名单）
→ hermes_test_exchange → hermes_save_exchange_credentials
```

卡片模板见 [exchange-cards.md](exchange-cards.md)。

### 3. 查询

| 工具 | 用途 |
|------|------|
| `hermes_get_klines` | K 线（binance/okx） |
| `hermes_get_balance` | 账户余额 |
| `hermes_get_positions` | 持仓 |
| `hermes_get_market_price` | 市价 |

### 4. 交易

交易前展示：`hermes_get_positions` + `hermes_get_balance`。

| action | 说明 |
|--------|------|
| `open_long` / `open_short` | 开仓（无同向持仓） |
| `add_long` / `add_short` | 加仓 |
| `close_long` / `close_short` | 全平（quantity 可省略） |
| `reduce_long` / `reduce_short` | 部分平仓（需 quantity） |

调用 `hermes_trade` 时 `confirmed` 必须为 `true`。

可选：`hermes_set_leverage`、`hermes_set_stop_loss`、`hermes_set_take_profit`。

## 支持的交易所

| ID | 交易 | K 线 |
|----|------|------|
| binance | ✅ | ✅ |
| okx | ✅ | ✅ |
| hyperliquid | ✅ | 暂不支持 |
| aster | ✅ | 暂不支持 |

## 卡片展示格式

向用户展示配置卡片时，使用结构化 Markdown：

```markdown
### [交易所名] 配置卡片
> 请按以下步骤在交易所官网操作，完成后将所需字段发给我。

**步骤 1** ...
**所需字段**：...
```
