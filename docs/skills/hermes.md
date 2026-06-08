# Hermes 交易 Skill

## 需求说明

做一套 Hermes 能使用作为交易的 skills。

## 要求

1. 必须经过合理的安全认证
2. 拥有所有交易所相关的接口，包括 K 线/持仓/开仓/加仓/减仓/平仓等等
3. 通过对话和用户获取交易所 API Key 等（以卡片形式引导客户自行获取）

## 实现索引

| 组件 | 路径 |
|------|------|
| Cursor Skill | [`.cursor/skills/hermes/SKILL.md`](../../.cursor/skills/hermes/SKILL.md) |
| 交易所配置卡片 | [`.cursor/skills/hermes/exchange-cards.md`](../../.cursor/skills/hermes/exchange-cards.md) |
| MCP Server | [`hermes/mcp-server/`](../../hermes/mcp-server/) |
| 后端 API | [`api/hermes.go`](../../api/hermes.go)、[`api/hermes_trader.go`](../../api/hermes_trader.go) |
| OKX K 线 | [`market/okx_klines.go`](../../market/okx_klines.go) |
| API 文档 | [`docs/swagger.json`](../swagger.json) |

## 快速开始

1. 启动 NOFX 后端（默认 `http://localhost:8080`）
2. 构建 MCP Server：

```bash
cd hermes/mcp-server && npm install && npm run build
```

3. 在 Cursor 中启用 MCP（见 [`.cursor/mcp.json`](../../.cursor/mcp.json)）
4. 对话中使用 Hermes Skill，先 `hermes_login` 再配置交易所与交易

详细说明见 [`hermes/mcp-server/README.md`](../../hermes/mcp-server/README.md)。
