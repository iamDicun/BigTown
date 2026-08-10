// Helper sinh dữ liệu cho load test: seed.sql + tokens.json.
//
// Đặt file này vào backend: backend/cmd/loadtest-gen/main.go
// rồi chạy TỪ THƯ MỤC backend/:
//
//	JWT_SECRET=<đúng secret của .env> go run ./cmd/loadtest-gen \
//	  -users=100 -rooms=10 -out=../loadtest
//
// Nó tái sử dụng CHÍNH security.GenerateToken của dự án => token chắc chắn
// tương thích với ParseToken ở Centrifuge OnConnecting (không tự chế lại JWT).
//
// UUID user/character/map sinh theo pattern cố định để seed.sql và tokens.json
// khớp nhau tuyệt đối, không lệch.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/security"
)

type tokenEntry struct {
	UserID string `json:"userId"`
	Token  string `json:"token"`
	Room   string `json:"room"` // map code VU này sẽ join (để đối chiếu, k6 cũng tự tính lại)
}

func userID(i int) string  { return fmt.Sprintf("00000000-0000-0000-0000-%012d", i) }
func charID(i int) string  { return fmt.Sprintf("00000000-0000-0000-0001-%012d", i) }
func mapID(i int) string   { return fmt.Sprintf("00000000-0000-0000-0002-%012d", i) }
func mapCode(i int) string { return fmt.Sprintf("loadtest-map-%02d", i) }

func main() {
	users := flag.Int("users", 100, "số user/character seed")
	rooms := flag.Int("rooms", 10, "số map/room (VU chia đều vào các room này)")
	ttlMin := flag.Int("ttl", 30, "TTL token (phút) — phải > thời lượng test")
	out := flag.String("out", ".", "thư mục xuất seed.sql và tokens.json")
	flag.Parse()

	secret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secret) == "" {
		fmt.Fprintln(os.Stderr, "Thiếu JWT_SECRET (phải trùng .env của backend)")
		os.Exit(1)
	}

	// ---- seed.sql ----
	var sb strings.Builder
	sb.WriteString("-- Seed load test. Chạy: psql <DSN> -f seed.sql\n")
	sb.WriteString("BEGIN;\n\n")

	sb.WriteString("-- maps\n")
	for r := 0; r < *rooms; r++ {
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO maps (id, code, name, tilemap_asset_key, tileset_asset_key, spawn_x, spawn_y, width, height, tile_size) "+
				"VALUES ('%s','%s','LoadTest Map %02d','lt_tilemap','lt_tileset',100,100,4000,4000,16) "+
				"ON CONFLICT (code) DO NOTHING;\n",
			mapID(r), mapCode(r), r))
	}
	sb.WriteString("\n-- users + characters (mỗi user 1 character; UNIQUE(user_id))\n")
	for i := 1; i <= *users; i++ {
		email := fmt.Sprintf("loadtest+%03d@bigtown.local", i)
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO app_user (id, full_name, email, role) VALUES ('%s','LoadTester %03d','%s','User') "+
				"ON CONFLICT (email) DO NOTHING;\n", userID(i), i, email))
		// character map_id để NULL cũng được; ở đây gán map theo vòng cho giống thực tế
		mIdx := (i - 1) % *rooms
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO characters (id, user_id, name, map_id, base_asset_key) "+
				"VALUES ('%s','%s','LT-%03d','%s','char_default') "+
				"ON CONFLICT (user_id) DO NOTHING;\n", charID(i), userID(i), i, mapID(mIdx)))
	}
	sb.WriteString("\nCOMMIT;\n")

	if err := os.WriteFile(filepath.Join(*out, "seed.sql"), []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ghi seed.sql:", err)
		os.Exit(1)
	}

	// ---- tokens.json ----
	ttl := time.Duration(*ttlMin) * time.Minute
	entries := make([]tokenEntry, 0, *users)
	for i := 1; i <= *users; i++ {
		tok, err := security.GenerateToken(userID(i), "User", secret, ttl)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sinh token:", err)
			os.Exit(1)
		}
		entries = append(entries, tokenEntry{
			UserID: userID(i),
			Token:  tok,
			Room:   mapCode((i - 1) % *rooms),
		})
	}
	buf, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(filepath.Join(*out, "tokens.json"), buf, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ghi tokens.json:", err)
		os.Exit(1)
	}

	fmt.Printf("OK: %d users, %d rooms, TTL %dm -> %s/{seed.sql,tokens.json}\n", *users, *rooms, *ttlMin, *out)
}
