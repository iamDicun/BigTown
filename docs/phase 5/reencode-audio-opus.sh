#!/usr/bin/env bash
#
# reencode-audio-opus.sh
# Re-encode nhạc/SFX sang Opus (.ogg) sao cho LUÔN nhỏ hơn bản mp3 hiện có.
#
# Vì sao cần: bản .ogg cũ trong opt/ được encode bằng Vorbis chất lượng cao
# nên NẶNG HƠN mp3. Trong khi audio.service.ts (detectAudioExt) lại ưu tiên ogg
# trên Chrome/Firefox/Edge -> đa số người dùng tải bản to hơn. Opus ~80k khắc phục:
#   intro  5.04MB(mp3) -> 4.69MB (hoặc 0.36MB nếu cắt loop)
#   dark_village 2.77 -> 2.04 | winter 2.07 -> 1.81 | village_adventure 1.63 -> 1.36
# Safari không đọc Ogg -> tự fallback sang mp3 (giữ nguyên mp3 làm fallback là ĐÚNG).
#
# Cách chạy (từ thư mục frontend/public/assets/sounds):
#   bash reencode-audio-opus.sh
#   bash reencode-audio-opus.sh --intro-loop     # cắt intro thành loop 24s (~0.36MB)
#
# Yêu cầu: ffmpeg có libopus (kiểm tra: ffmpeg -encoders | grep opus)

set -euo pipefail

# --- Thư mục nguồn & đích ---
# Nguồn ưu tiên: file gốc ít nén nhất. Nếu bạn còn file gốc ở thư mục hiện tại,
# script dùng nó; nếu không, dùng chính bản mp3 trong opt/ làm nguồn.
SRC_DIR="."          # nơi chứa mp3 gốc (intro.mp3, winter.mp3, ...)
OUT_DIR="opt"        # nơi app đang đọc (đã có sẵn *.mp3, ta ghi đè *.ogg)
INTRO_LOOP=0
[[ "${1:-}" == "--intro-loop" ]] && INTRO_LOOP=1

mkdir -p "$OUT_DIR"

# Danh sách file: "tên|loại|bitrate|kênh"
#   music = nền bản đồ/intro (stereo), sfx = âm ngắn (mono)
MUSIC=("intro" "dark_village" "winter" "village_adventure")
SFX=("click" "f1" "f2" "f3" "f4" "f5" "f6" "f7")

MUSIC_BITRATE=80    # kbps opus cho nhạc nền (đã kiểm chứng < mp3)
SFX_BITRATE=56      # kbps opus mono cho SFX

human() { awk -v b="$1" 'BEGIN{printf "%.2fMB", b/1048576}'; }

# Tìm file nguồn tốt nhất cho 1 tên (ưu tiên SRC_DIR, fallback OUT_DIR/mp3)
find_src() {
  local name="$1"
  if [[ -f "$SRC_DIR/$name.mp3" ]]; then echo "$SRC_DIR/$name.mp3"; return; fi
  if [[ -f "$SRC_DIR/$name.wav" ]]; then echo "$SRC_DIR/$name.wav"; return; fi
  if [[ -f "$OUT_DIR/$name.mp3" ]]; then echo "$OUT_DIR/$name.mp3"; return; fi
  echo ""   # không tìm thấy
}

# Encode 1 file; nếu ogg >= mp3 thì tự hạ bitrate tối đa 2 lần cho tới khi nhỏ hơn
encode_guarded() {
  local name="$1" kind="$2" bitrate="$3"
  local src; src="$(find_src "$name")"
  [[ -z "$src" ]] && { echo "  ! bỏ qua $name (không có nguồn)"; return; }

  local out="$OUT_DIR/$name.ogg"
  local mp3="$OUT_DIR/$name.mp3"
  local ch_args=(); [[ "$kind" == "sfx" ]] && ch_args=(-ac 1)

  local try=0
  while (( try < 3 )); do
    ffmpeg -y -loglevel error -i "$src" \
      -c:a libopus -b:a "${bitrate}k" -vbr on -ar 48000 "${ch_args[@]}" "$out"

    local ogg_sz mp3_sz
    ogg_sz=$(stat -c%s "$out")
    mp3_sz=$(stat -c%s "$mp3" 2>/dev/null || echo 999999999)

    if (( ogg_sz < mp3_sz )); then
      printf "  ✓ %-20s opus %3dk  %10s  (mp3 %s)\n" "$name" "$bitrate" "$(human $ogg_sz)" "$(human $mp3_sz)"
      return
    fi
    bitrate=$(( bitrate - 16 )); try=$(( try + 1 ))
    echo "    (ogg còn to hơn mp3, hạ xuống ${bitrate}k thử lại...)"
  done
  printf "  ⚠ %-20s vẫn chưa nhỏ hơn mp3 — cân nhắc cắt loop hoặc dùng mp3 cho file này\n" "$name"
}

echo "=== NHẠC NỀN (opus ${MUSIC_BITRATE}k stereo) ==="
for name in "${MUSIC[@]}"; do
  if [[ "$name" == "intro" && "$INTRO_LOOP" == "1" ]]; then
    src="$(find_src intro)"
    # Cắt 24s kể từ giây 8, fade in/out để loop mượt. ~0.36MB.
    ffmpeg -y -loglevel error -ss 8 -t 24 -i "$src" \
      -c:a libopus -b:a 96k -ar 48000 \
      -af "afade=t=in:st=0:d=1.5,afade=t=out:st=22.5:d=1.5" "$OUT_DIR/intro.ogg"
    # cũng tạo mp3 loop cùng độ dài để Safari fallback khớp
    ffmpeg -y -loglevel error -ss 8 -t 24 -i "$src" \
      -c:a libmp3lame -b:a 112k -ar 44100 \
      -af "afade=t=in:st=0:d=1.5,afade=t=out:st=22.5:d=1.5" "$OUT_DIR/intro.mp3"
    printf "  ✓ %-20s LOOP 24s  %10s  (thay cho ~5MB)\n" "intro" "$(human $(stat -c%s "$OUT_DIR/intro.ogg"))"
  else
    encode_guarded "$name" music "$MUSIC_BITRATE"
  fi
done

echo ""
echo "=== SFX (opus ${SFX_BITRATE}k mono) ==="
for name in "${SFX[@]}"; do
  encode_guarded "$name" sfx "$SFX_BITRATE"
done

echo ""
echo "=== TỔNG KẾT thư mục $OUT_DIR ==="
total_ogg=0; total_mp3=0
for name in "${MUSIC[@]}" "${SFX[@]}"; do
  o=$(stat -c%s "$OUT_DIR/$name.ogg" 2>/dev/null || echo 0)
  m=$(stat -c%s "$OUT_DIR/$name.mp3" 2>/dev/null || echo 0)
  total_ogg=$(( total_ogg + o )); total_mp3=$(( total_mp3 + m ))
done
echo "  Tổng ogg (Chrome/FF/Edge tải): $(human $total_ogg)"
echo "  Tổng mp3 (Safari fallback):    $(human $total_mp3)"
echo ""
echo "Xong. Nghe thử trong opt/ rồi commit. Không cần sửa code:"
echo "resolveSound() vẫn trỏ opt/<name>.<ext>, giờ .ogg đã nhỏ hơn .mp3."
