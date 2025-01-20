package database

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")
	logger := zap.NewNop()

	// 测试创建数据库
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// 验证数据库文件是否创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// 验证表是否创建
	var count int
	err = db.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' AND name='file_records'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 1 {
		t.Error("file_records table was not created")
	}

	// 验证索引是否创建
	rows, err := db.db.Query(`
		SELECT name FROM sqlite_master 
		WHERE type='index' AND tbl_name='file_records'
	`)
	if err != nil {
		t.Fatalf("failed to check indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan index name: %v", err)
		}
		indexes[name] = true
	}

	expectedIndexes := []string{
		"idx_file_records_status",
		"idx_file_records_source_path",
		"idx_file_records_source_path_unique",
	}

	for _, idx := range expectedIndexes {
		if !indexes[idx] {
			t.Errorf("index %s was not created", idx)
		}
	}

	// 测试错误情况
	t.Run("invalid path", func(t *testing.T) {
		_, err := New("", logger)
		if err == nil {
			t.Error("expected error for invalid path")
		}
	})
}
