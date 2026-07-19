package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	"backend/internal/platform/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresDB struct {
	DB *sql.DB
}

func NewPostgresDB(cfg config.DatabaseConfig) *PostgresDB {
	query := url.Values{}
	query.Set("sslmode", cfg.SSLMode)
	query.Set("default_query_exec_mode", "simple_protocol")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?%s", url.QueryEscape(cfg.User), url.QueryEscape(cfg.Password), cfg.Host, cfg.Port, cfg.Name, query.Encode())

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("Lỗi khởi tạo driver: ", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		log.Fatal("Lỗi kết nối DB: ", err)
	}
	fmt.Println("Kết nối PostgreSQL thành công!")

	return &PostgresDB{DB: db}
}

func (p *PostgresDB) Close() {
	if p.DB != nil {
		p.DB.Close()
	}
}
