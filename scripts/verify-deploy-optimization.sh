#!/usr/bin/env bash
# 部署优化改动本地验证（无需完整 Docker 构建）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
pass=0
fail=0

ok() { echo -e "${GREEN}✓${NC} $1"; pass=$((pass + 1)); }
bad() { echo -e "${RED}✗${NC} $1"; fail=$((fail + 1)); }

echo "=== NOFX 部署优化验证 ==="

# 1. start.sh 语法
if bash -n start.sh 2>/dev/null; then
  ok "start.sh 语法检查"
else
  bad "start.sh 语法检查"
fi

# 2. docker compose 配置
if command -v docker >/dev/null 2>&1; then
  if docker compose config >/dev/null 2>&1; then
    ok "docker compose config"
  else
    bad "docker compose config"
  fi
else
  echo "  (跳过 docker compose：未安装 docker)"
fi

# 3. 必需文件
for f in docker/Dockerfile.ta-lib-base docker/Dockerfile.backend docker/Dockerfile.frontend; do
  if [[ -f "$f" ]]; then
    ok "存在 $f"
  else
    bad "缺失 $f"
  fi
done

# 4. backend Dockerfile 含 default / fast target
if grep -q 'AS default' docker/Dockerfile.backend && grep -q 'AS fast' docker/Dockerfile.backend; then
  ok "Dockerfile.backend 含 default 与 fast target"
else
  bad "Dockerfile.backend 缺少 target"
fi

# 5. CI workflow 含变更检测与 ta-lib-base job
if grep -q 'detect-changes' .github/workflows/docker-build.yml && grep -q 'build-ta-lib-base' .github/workflows/docker-build.yml; then
  ok "CI 含 detect-changes 与 build-ta-lib-base"
else
  bad "CI workflow 结构不完整"
fi

# 6. 模拟 detect-changes（仅 web 变更 -> 不构建 backend）
simulate_detect() {
  local files="$1"
  local expect_backend="$2"
  local expect_frontend="$3"
  BUILD_BACKEND=false
  BUILD_FRONTEND=false
  if echo "$files" | grep -E -q '(^docker/Dockerfile\.backend$|^docker/Dockerfile\.ta-lib-base$|^go\.mod$|^go\.sum$|\.go$|^\.github/workflows/docker-build\.yml$)'; then
    BUILD_BACKEND=true
  fi
  if echo "$files" | grep -E -q '(^docker/Dockerfile\.frontend$|^web/|^nginx/|^\.github/workflows/docker-build\.yml$)'; then
    BUILD_FRONTEND=true
  fi
  [[ "$BUILD_BACKEND" == "$expect_backend" ]] && [[ "$BUILD_FRONTEND" == "$expect_frontend" ]]
}

if simulate_detect "web/src/App.tsx" false true; then
  ok "detect: 仅 web 变更 -> 仅 frontend"
else
  bad "detect: 仅 web 变更"
fi

if simulate_detect "main.go" true false; then
  ok "detect: 仅 go 变更 -> 仅 backend"
else
  bad "detect: 仅 go 变更"
fi

if simulate_detect "docker/Dockerfile.ta-lib-base" true false; then
  ok "detect: ta-lib-base 变更 -> backend"
else
  bad "detect: ta-lib-base 变更"
fi

# 7. update 帮助文案
if grep -q 'update \[--build\]' start.sh; then
  ok "start.sh update 快速/重建 说明"
else
  bad "start.sh update 说明"
fi

echo ""
echo "结果: ${pass} 通过, ${fail} 失败"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
echo "全部检查通过。"
