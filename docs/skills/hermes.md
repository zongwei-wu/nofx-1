# Hermes 自主交易 Skill

## 概述

Hermes 是 NOFX 的**独立**自主交易体系，与 Web「AI 交易员」解耦：

- 策略由 **Hermes Runner** + **`prompts/hermes.txt`** 自主决策
- 用户授权 `autonomous_enabled` 后，Runner 下单**无需**逐笔 `confirmed`
- Cursor 对话内手动 `hermes_trade` **仍须** `confirmed: true`

## 实现索引

| 组件 | 路径 |
|------|------|
| Cursor Skill | [`.cursor/skills/hermes/SKILL.md`](../../.cursor/skills/hermes/SKILL.md) |
| 交易所配置卡片 | [`.cursor/skills/hermes/exchange-cards.md`](../../.cursor/skills/hermes/exchange-cards.md) |
| MCP Server | [`hermes/mcp-server/`](../../hermes/mcp-server/) |
| Runner / Manager | [`hermes/runner.go`](../../hermes/runner.go)、[`hermes/manager.go`](../../hermes/manager.go) |
| 设置 API | [`api/hermes_settings.go`](../../api/hermes_settings.go) |
| 交易 API | [`api/hermes.go`](../../api/hermes.go)、[`api/hermes_trader.go`](../../api/hermes_trader.go) |
| 策略 Prompt | [`prompts/hermes.txt`](../../prompts/hermes.txt) |
| 权限 | [`config/permissions.go`](../../config/permissions.go) `FeatureHermes` |
| 设置表 | `hermes_settings`（见 [`config/database.go`](../../config/database.go)） |
| API 文档 | [`docs/swagger.json`](../swagger.json) |

## 自主交易工作流

```
hermes_login
→ hermes_require_exchange_setup
→ hermes_save_exchange_credentials + hermes_list_models
→ hermes_update_settings（exchange_id、ai_model_id）
→ hermes_enable_autonomous(enabled: true)   # 用户明确授权
→ hermes_start_runner
→ hermes_get_runner_status / hermes_get_decisions
```

停止：`hermes_stop_runner`

## 数据表（hermes_settings）

```sql
CREATE TABLE IF NOT EXISTS hermes_settings (
  user_id TEXT PRIMARY KEY,
  runner_enabled INTEGER NOT NULL DEFAULT 0,
  autonomous_enabled INTEGER NOT NULL DEFAULT 0,
  exchange_id TEXT NOT NULL DEFAULT '',
  ai_model_id TEXT NOT NULL DEFAULT '',
  scan_interval_minutes INTEGER NOT NULL DEFAULT 5,
  system_prompt_template TEXT NOT NULL DEFAULT 'hermes',
  btc_eth_leverage INTEGER NOT NULL DEFAULT 5,
  altcoin_leverage INTEGER NOT NULL DEFAULT 3,
  trading_coins TEXT NOT NULL DEFAULT '',
  initial_balance REAL NOT NULL DEFAULT 0,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

决策日志目录：`decision_logs/hermes/{user_id}/`

## 快速开始

1. 启动 NOFX 后端
2. 构建 MCP：`cd hermes/mcp-server && npm install && npm run build`
3. 在 Cursor 启用 MCP（[`.cursor/mcp.json`](../../.cursor/mcp.json)）
4. 使用 Hermes Skill 完成配置与启停

详细 MCP 说明见 [`hermes/mcp-server/README.md`](../../hermes/mcp-server/README.md)。
