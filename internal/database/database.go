package database

import (
	"database/sql"
	"embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

//go:embed schema.sql
var schemaFS embed.FS

type Database struct {
	db     *sql.DB
	logger *zap.Logger
}

func New(dbPath string, logger *zap.Logger) (*Database, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 读取并执行schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("execute schema: %w", err)
	}

	return &Database{
		db:     db,
		logger: logger,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
