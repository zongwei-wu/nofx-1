#!/bin/bash
# NOFX API Key 端到端验证脚本
# 用法: ./test-apikey.sh <email> <password> [base_url]

set -e

BASE_URL="${3:-http://localhost:8080}"
EMAIL="$1"
PASSWORD="$2"

if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ]; then
  echo "用法: $0 <email> <password> [base_url]"
  exit 1
fi

echo "═══ NOFX API Key 验证测试 ═══"
echo ""

# 1. 登录
echo "① 登录..."
LOGIN_RES=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "   $LOGIN_RES"

# 检查是否需要 OTP
if echo "$LOGIN_RES" | grep -q "requires_otp"; then
  USER_ID=$(echo "$LOGIN_RES" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
  echo ""
  echo "   需要 OTP 验证"
  echo -n "   请输入 Google Authenticator 验证码: "
  read OTP_CODE
  
  OTP_RES=$(curl -s -X POST "$BASE_URL/api/verify-otp" \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"$USER_ID\",\"otp_code\":\"$OTP_CODE\"}")
  echo "   $OTP_RES"
  JWT=$(echo "$OTP_RES" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
else
  JWT=$(echo "$LOGIN_RES" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$JWT" ]; then
  echo "❌ 登录失败，无法获取 JWT"
  exit 1
fi
echo "   ✅ JWT 获取成功"
echo ""

# 2. 创建 API Key
echo "② 创建 API Key..."
CREATE_RES=$(curl -s -X POST "$BASE_URL/api/api-keys" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"name":"测试Key"}')
echo "   $CREATE_RES"

API_KEY=$(echo "$CREATE_RES" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
KEY_ID=$(echo "$CREATE_RES" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$API_KEY" ]; then
  echo "❌ 创建 API Key 失败"
  exit 1
fi
echo "   ✅ API Key: ${API_KEY:0:20}..."
echo ""

# 3. 使用 API Key 访问 MCP 端点
echo "③ 使用 API Key 访问 MCP 端点..."

# 3a. 获取交易所列表
echo -n "   获取交易所列表... "
EXCHANGES=$(curl -s "$BASE_URL/api/mcp/exchanges" \
  -H "Authorization: ApiKey $API_KEY")
echo "$EXCHANGES"

# 3b. 获取 AI 模型列表
echo -n "   获取模型列表... "
MODELS=$(curl -s "$BASE_URL/api/mcp/models" \
  -H "Authorization: ApiKey $API_KEY")
echo "$MODELS"

# 3c. 获取 Hermes 状态
echo -n "   获取 Hermes 状态... "
STATUS=$(curl -s "$BASE_URL/api/mcp/hermes/status" \
  -H "Authorization: ApiKey $API_KEY")
echo "$STATUS"

echo ""
echo "④ 清理：撤销 API Key..."
curl -s -X DELETE "$BASE_URL/api/api-keys/$KEY_ID" \
  -H "Authorization: Bearer $JWT"
echo ""

# 4. 验证已撤销的 Key 不可用
echo ""
echo "⑤ 验证已撤销的 Key 不可用..."
REVOKED=$(curl -s "$BASE_URL/api/mcp/exchanges" \
  -H "Authorization: ApiKey $API_KEY")
echo "   $REVOKED"
if echo "$REVOKED" | grep -q "无效"; then
  echo "   ✅ 撤销生效，已拒绝"
else
  echo "   ⚠️  撤销可能未生效"
fi

echo ""
echo "═══ 测试完成 ═══"
