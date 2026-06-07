#!/usr/bin/env bash
# 清空测试用交易记录（保留用户、交易员、API 配置与跟单关注设置）
# 用法:
#   ./scripts/reset-test-records.sh              # 交互确认
#   ./scripts/reset-test-records.sh --backup -y  # 先备份再执行
#   ./scripts/reset-test-records.sh --user-id <id>  # 仅清指定用户 DB 记录

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DB_PATH="${NOFX_DB_PATH:-config.db}"
DECISION_LOGS_DIR="${NOFX_DECISION_LOGS_DIR:-decision_logs}"
COPY_CACHE_DIR="${NOFX_COPY_CACHE_DIR:-data/copy-trading}"

DO_BACKUP=false
ASSUME_YES=false
USER_ID=""

usage() {
  cat <<'EOF'
测试初始化：清空所有交易记录

选项:
  --backup          执行前备份 config.db、decision_logs、data/copy-trading
  -y, --yes         跳过确认提示
  --user-id <id>    仅清空指定用户的 DB 记录（decision_logs 仍全部清空）
  --db <path>       数据库路径（默认 config.db）
  -h, --help        显示帮助

保留: users, traders, ai_models, exchanges, copy_trade_config, 币种偏好等
清空: copy_trade_records, copy_trade_runs/events, copy_trade_ai_decisions,
      decision_logs/, data/copy-trading/
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --backup) DO_BACKUP=true; shift ;;
    -y|--yes) ASSUME_YES=true; shift ;;
    --user-id) USER_ID="${2:-}"; shift 2 ;;
    --db) DB_PATH="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "未知参数: $1" >&2; usage; exit 1 ;;
  esac
done

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "错误: 需要 sqlite3 命令" >&2
  exit 1
fi

if [[ ! -f "$DB_PATH" ]]; then
  echo "错误: 数据库不存在: $DB_PATH" >&2
  exit 1
fi

if command -v lsof >/dev/null 2>&1 && lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "警告: 端口 8080 有进程监听，建议先停止 nofx 后端再执行清空。"
  echo "      Docker: docker compose stop nofx"
fi

echo "将清空以下交易记录（项目根: $ROOT_DIR）:"
echo "  DB: $DB_PATH"
if [[ -n "$USER_ID" ]]; then
  echo "  范围: 仅用户 $USER_ID（DB）；decision_logs 仍全部清空"
else
  echo "  范围: 全部用户"
fi
echo "  目录: $DECISION_LOGS_DIR/*"
echo "  目录: $COPY_CACHE_DIR/*"
echo ""
echo "保留: 用户/交易员/API/跟单关注配置 (copy_trade_config)"
echo ""

if [[ "$ASSUME_YES" != true ]]; then
  read -r -p "确认继续? [y/N] " ans
  case "$ans" in
    y|Y|yes|YES) ;;
    *) echo "已取消"; exit 0 ;;
  esac
fi

if [[ "$DO_BACKUP" == true ]]; then
  STAMP="$(date +%Y%m%d_%H%M%S)"
  BK_DB="${DB_PATH}.bak.${STAMP}"
  cp "$DB_PATH" "$BK_DB"
  echo "已备份数据库 -> $BK_DB"
  BK_TAR="backup-test-records-${STAMP}.tar.gz"
  tar -czf "$BK_TAR" "$DECISION_LOGS_DIR" "$COPY_CACHE_DIR" 2>/dev/null || true
  if [[ -f "$BK_TAR" ]]; then
    echo "已备份目录 -> $BK_TAR"
  fi
fi

run_sql() {
  if [[ -n "$USER_ID" ]]; then
    sqlite3 "$DB_PATH" <<SQL
DELETE FROM copy_trade_ai_decisions WHERE user_id = '$USER_ID';
DELETE FROM copy_trade_run_events WHERE run_id IN (SELECT id FROM copy_trade_runs WHERE user_id = '$USER_ID');
DELETE FROM copy_trade_runs WHERE user_id = '$USER_ID';
DELETE FROM copy_trade_records WHERE user_id = '$USER_ID';
VACUUM;
SQL
  else
    sqlite3 "$DB_PATH" <<'SQL'
DELETE FROM copy_trade_ai_decisions;
DELETE FROM copy_trade_run_events;
DELETE FROM copy_trade_runs;
DELETE FROM copy_trade_records;
VACUUM;
SQL
  fi
}

run_sql
echo "已清空数据库交易记录表"

if [[ -d "$DECISION_LOGS_DIR" ]]; then
  find "$DECISION_LOGS_DIR" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
  echo "已清空 $DECISION_LOGS_DIR"
fi
mkdir -p "$DECISION_LOGS_DIR"

if [[ -d "$COPY_CACHE_DIR" ]]; then
  find "$COPY_CACHE_DIR" -mindepth 1 -maxdepth 1 ! -name '.gitkeep' -exec rm -rf {} +
  echo "已清空 $COPY_CACHE_DIR（保留 .gitkeep）"
fi
mkdir -p "$COPY_CACHE_DIR"
touch "$COPY_CACHE_DIR/.gitkeep" 2>/dev/null || true

echo ""
echo "完成。请重启后端后验证:"
echo "  docker compose up -d nofx"
echo "  或本地: go run ."
