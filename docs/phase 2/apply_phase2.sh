#!/usr/bin/env bash
# =============================================================================
# BigTown — Apply Phase 2 DB changes (migrate + seed), idempotent.
#
# Dùng cho 2 tình huống:
#   1) DB đang chạy (local/Render): áp migrate + seed để "kéo" DB lên Phase 2.
#   2) Dựng DB integration/CI từ đầu (--fresh): chạy schema.sql + seed.sql gốc
#      trước, rồi migrate + seed Phase 2. Kết quả là 1 DB đầy đủ để test.
#
# Kết nối: ưu tiên biến DATABASE_URL; nếu không có, ghép từ PG* / mặc định
# khớp docker-compose (postgres/postgres @ localhost:5433/app_db).
#
# Ví dụ:
#   ./apply_phase2.sh                          # áp lên DB (DATABASE_URL hoặc mặc định)
#   DATABASE_URL=postgres://u:p@host/db ./apply_phase2.sh
#   ./apply_phase2.sh --fresh                  # dựng lại toàn bộ (CI/integration)
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Đường dẫn schema/seed gốc (khi chạy từ backend/internal/database/ thì cùng thư mục).
SCHEMA_SQL="${SCHEMA_SQL:-$SCRIPT_DIR/schema.sql}"
SEED_SQL="${SEED_SQL:-$SCRIPT_DIR/seed.sql}"
MIGRATE_SQL="$SCRIPT_DIR/phase2_migrate.sql"
SEED_PHASE2_SQL="$SCRIPT_DIR/phase2_seed.sql"

# Chuỗi kết nối.
if [[ -z "${DATABASE_URL:-}" ]]; then
  PGHOST="${PGHOST:-localhost}"; PGPORT="${PGPORT:-5433}"
  PGUSER="${PGUSER:-postgres}"; PGPASSWORD="${PGPASSWORD:-postgres}"
  PGDATABASE="${PGDATABASE:-app_db}"
  export PGPASSWORD
  DATABASE_URL="postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${PGDATABASE}"
fi

run() { echo ">> psql -f $1"; psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$1"; }

FRESH=0
[[ "${1:-}" == "--fresh" ]] && FRESH=1

if [[ "$FRESH" == "1" ]]; then
  echo "== FRESH: schema + seed gốc =="
  [[ -f "$SCHEMA_SQL" ]] && run "$SCHEMA_SQL" || echo "!! bỏ qua schema.sql (không thấy $SCHEMA_SQL)"
  [[ -f "$SEED_SQL" ]]   && run "$SEED_SQL"   || echo "!! bỏ qua seed.sql (không thấy $SEED_SQL)"
fi

echo "== Phase 2 migrate =="
run "$MIGRATE_SQL"
echo "== Phase 2 seed =="
run "$SEED_PHASE2_SQL"

echo "✅ Done. Kiểm tra:"
echo "   psql \"\$DATABASE_URL\" -c \"SELECT code, music_asset_key FROM maps;\""
echo "   psql \"\$DATABASE_URL\" -c \"SELECT count(*) FROM items WHERE type='decoration';\""
echo "   psql \"\$DATABASE_URL\" -c \"SELECT count(*) FROM map_npc_spawns;\""
