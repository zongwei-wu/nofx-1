# 外部 Hermes 与 NOFX 网关

NOFX 不再内建 Hermes Runner。自主交易由**自建 Hermes 服务**完成；NOFX 仅提供**交易所网关 API**。

## 架构

```
自建 Hermes（Agent 循环 + 策略）
    ↓ API Key
NOFX /api/gateway/*
    ↓ exchanges 表凭证
交易所（Binance/OKX/...）
```

## 文档索引

| 组件 | 路径 |
|------|------|
| Gateway API 接入 | [`docs/gateway-api.md`](../gateway-api.md) |
| Go SDK | [`gateway/client/`](../../gateway/client/) |
| Swagger | [`docs/swagger.json`](../swagger.json) |
| API Key 管理 | Web → API Keys 页面 |

## 快速接入

1. NOFX Web 配置交易所 + 生成 API Key
2. Hermes 服务：`client.New(NOFX_GATEWAY_URL, NOFX_API_KEY)`
3. 调用 `ListExchanges` → `GetBalance` → `Trade`

**禁止**调用 `/api/traders/*`（那是 Web AI 交易员，与 Hermes 无关）。
