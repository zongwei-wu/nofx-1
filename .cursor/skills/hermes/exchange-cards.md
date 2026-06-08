# Hermes 交易所配置卡片

通过对话引导用户自行在交易所官网创建 API Key，再将凭证提交给 Hermes（经 RSA 加密入库）。

**白名单 IP**：请先调用 `hermes_get_server_ip` 获取 NOFX 服务器公网 IP。

---

## binance

**交易所**：Binance 合约（USDT 永续）

**步骤 1**：登录 [Binance](https://www.binance.com) → 用户中心 → API 管理

**步骤 2**：创建 API Key，权限勾选「启用现货与杠杆交易」和「启用合约」

**步骤 3**：IP 访问限制选择「限制访问受信任的 IP」，填入服务器公网 IP

**步骤 4**：妥善保存 API Key 与 Secret Key（Secret 仅显示一次）

**所需字段**：
- `api_key`
- `secret_key`
- `testnet`（可选，模拟盘为 true）

**提交顺序**：收集字段 → `hermes_test_exchange` 测试 → `hermes_save_exchange_credentials` 保存

---

## okx

**交易所**：OKX 合约（USDT 永续）

**步骤 1**：登录 [OKX](https://www.okx.com) → 个人中心 → API

**步骤 2**：创建 API Key，权限选择「交易」，绑定 IP 白名单

**步骤 3**：记录 API Key、Secret Key 和 **Passphrase**（创建时自行设置）

**所需字段**：
- `api_key`
- `secret_key`
- `passphrase`（必填）
- `testnet`（可选）

**提交顺序**：收集字段 → `hermes_test_exchange` → `hermes_save_exchange_credentials`

---

## hyperliquid

**交易所**：Hyperliquid

**步骤 1**：准备主钱包地址（存放资金）与 Agent 私钥（用于签名，建议近零余额）

**步骤 2**：参考 [Hyperliquid API 钱包文档](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/nonces-and-api-wallets) 创建 Agent Wallet

**步骤 3**：私钥分两段输入（降低剪贴板泄露风险）：先输入前半段，再输入后半段

**所需字段**：
- `api_key`（Agent 私钥）
- `hyperliquid_wallet_addr`（主钱包地址）
- `testnet`（可选）

**提交顺序**：收集字段 → `hermes_test_exchange` → `hermes_save_exchange_credentials`

---

## aster

**交易所**：Aster DEX

**步骤 1**：在 Aster 平台获取 User、Signer 地址

**步骤 2**：生成 Signer 私钥（建议分两段输入）

**步骤 3**：确认 API 权限与网络环境

**所需字段**：
- `aster_user`
- `aster_signer`
- `aster_private_key`

**提交顺序**：收集字段 → `hermes_test_exchange` → `hermes_save_exchange_credentials`
