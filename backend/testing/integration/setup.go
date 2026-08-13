package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"backend/internal/app"
	"backend/internal/platform/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB     *sql.DB
	testDBOnce sync.Once
)

func GetTestDB(t *testing.T) *sql.DB {
	testDBOnce.Do(func() {
		dbURL := os.Getenv("TEST_DATABASE_URL")
		if dbURL == "" {
			dbURL = "postgres://test_user:test_pass@localhost:5434/bigtown_test?sslmode=disable"
		}

		var err error
		testDB, err = sql.Open("pgx", dbURL)
		if err != nil {
			t.Fatalf("failed to open test database: %v", err)
		}

		// Wait up to 10s for Postgres to be ready
		for i := 0; i < 10; i++ {
			if err = testDB.Ping(); err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			t.Fatalf("test database ping failed: %v", err)
		}

		// Auto-migrate schema & seed if tables don't exist
		ensureSchemaAndSeed(t, testDB)
	})

	return testDB
}

func ensureSchemaAndSeed(t *testing.T, db *sql.DB) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'app_user')").Scan(&exists)
	if err == nil && exists {
		return
	}

	// Reset schema to ensure clean slate
	_, _ = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")

	// Read schema.sql & seed.sql relative to current file or working dir
	baseDir := findProjectBackendDir()
	schemaPath := filepath.Join(baseDir, "internal", "database", "schema.sql")
	seedPath := filepath.Join(baseDir, "internal", "database", "seed.sql")

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema.sql from %s: %v", schemaPath, err)
	}

	if _, err := db.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("failed to execute schema.sql: %v", err)
	}

	seedBytes, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("failed to read seed.sql from %s: %v", seedPath, err)
	}

	if _, err := db.Exec(string(seedBytes)); err != nil {
		t.Fatalf("failed to execute seed.sql: %v", err)
	}
}

func findProjectBackendDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func TruncateTables(t *testing.T, db *sql.DB) {
	// Re-verify tables exist before truncating
	ensureSchemaAndSeed(t, db)

	query := `TRUNCATE app_user, credential, refresh_token, token_blacklist, user_identities, characters, chat_messages, map_placements RESTART IDENTITY CASCADE;`
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("failed to truncate test database tables: %v", err)
	}
}

func NewTestApp(db *sql.DB) *app.App {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Auth: config.AuthConfig{
			JWTSecret: "test-jwt-secret-12345",
		},
		Game: config.GameConfig{
			DefaultMapCode: "village_adventure",
			StartingCoins:  1000,
		},
		Web: config.WebConfig{
			AllowedOrigins: []string{"*"},
		},
	}

	container := &app.Container{
		Config: cfg,
		DB:     db,
	}

	return app.New(container)
}

func DoRequest(router http.Handler, method string, path string, body any, token string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
