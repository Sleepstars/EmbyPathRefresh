package processor

import (
	"database/sql"
	"github.com/sleepstars/embypathrefresh/internal/model"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessor_ProcessFile(t *testing.T) {
	// 设置测试环境
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	
	// 创建测试目录
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建测试文件
	testFilePath := filepath.Join(sourceDir, "test.mkv")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建测试数据库
	embyDBPath := filepath.Join(tmpDir, "library.db")
	embyDB, err := sql.Open("sqlite3", embyDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer embyDB.Close()

	// 创建测试表
	_, err = embyDB.Exec(`
		CREATE TABLE MediaItems (
			Id INTEGER PRIMARY KEY,
			Path TEXT
		);
		INSERT INTO MediaItems (Path) VALUES (?);
	`, testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	// 创建应用数据库
	appDBPath := filepath.Join(tmpDir, "app.db")
	appDB, err := sql.Open("sqlite3", appDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer appDB.Close()

	// 创建应用表
	_, err = appDB.Exec(`
		CREATE TABLE file_records (
			id INTEGER PRIMARY KEY,
			source_path TEXT,
			target_path TEXT,
			modified_time DATETIME,
			processed_time DATETIME,
			delete_scheduled DATETIME,
			status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	// 创建处理器
	logger := zap.NewNop()
	proc, err := New(embyDBPath, appDBPath, sourceDir, targetDir, 24*time.Hour, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()

	// 测试文件处理
	record := &model.FileRecord{
		SourcePath:   testFilePath,
		ModifiedTime: time.Now(),
		Status:      "pending",
	}

	if err := proc.ProcessFile(record); err != nil {
		t.Fatal(err)
	}

	// 验证文件是否已移动
	expectedTargetPath := filepath.Join(targetDir, "test.mkv")
	if _, err := os.Stat(expectedTargetPath); os.IsNotExist(err) {
		t.Error("file was not moved to target location")
	}

	// 验证数据库记录
	var path string
	err = embyDB.QueryRow("SELECT Path FROM MediaItems WHERE Path = ?", expectedTargetPath).Scan(&path)
	if err != nil {
		t.Error("path was not updated in emby database")
	}

	var count int
	err = appDB.QueryRow("SELECT COUNT(*) FROM file_records WHERE source_path = ? AND status = 'processed'", 
		testFilePath).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Error("file record was not created properly")
	}

	// 测试重复处理同一文件
	t.Run("duplicate file", func(t *testing.T) {
		if err := proc.ProcessFile(record); err != nil {
			t.Fatal(err)
		}
		var duplicateCount int
		err = appDB.QueryRow("SELECT COUNT(*) FROM file_records WHERE source_path = ?", 
			testFilePath).Scan(&duplicateCount)
		if err != nil {
			t.Fatal(err)
		}
		if duplicateCount != 1 {
			t.Error("duplicate record was created")
		}
	})
}

func TestProcessor_CleanupFiles(t *testing.T) {
	// 设置测试环境
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建测试文件
	testFile1 := filepath.Join(sourceDir, "test1.mkv")
	testFile2 := filepath.Join(sourceDir, "test2.mkv")
	for _, file := range []string{testFile1, testFile2} {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 创建数据库
	embyDBPath := filepath.Join(tmpDir, "library.db")
	embyDB, err := sql.Open("sqlite3", embyDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer embyDB.Close()

	_, err = embyDB.Exec(`CREATE TABLE MediaItems (Id INTEGER PRIMARY KEY, Path TEXT);`)
	if err != nil {
		t.Fatal(err)
	}

	// 创建应用数据库
	appDBPath := filepath.Join(tmpDir, "app.db")
	appDB, err := sql.Open("sqlite3", appDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer appDB.Close()

	now := time.Now()
	pastTime := now.Add(-2 * time.Hour)
	futureTime := now.Add(2 * time.Hour)

	// 创建应用表并插入测试数据
	_, err = appDB.Exec(`
		CREATE TABLE file_records (
			id INTEGER PRIMARY KEY,
			source_path TEXT,
			target_path TEXT,
			modified_time DATETIME,
			processed_time DATETIME,
			delete_scheduled DATETIME,
			status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
		INSERT INTO file_records (source_path, target_path, modified_time, processed_time, delete_scheduled, status, created_at, updated_at)
		VALUES 
		(?, ?, ?, ?, ?, 'processed', ?, ?),
		(?, ?, ?, ?, ?, 'processed', ?, ?);
	`, testFile1, filepath.Join(targetDir, "test1.mkv"), 
	   now, now, pastTime, now, now,
	   testFile2, filepath.Join(targetDir, "test2.mkv"),
	   now, now, futureTime, now, now)
	if err != nil {
		t.Fatal(err)
	}

	// 创建处理器
	logger := zap.NewNop()
	proc, err := New(embyDBPath, appDBPath, sourceDir, targetDir, 24*time.Hour, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()

	// 运行清理
	if err := proc.CleanupFiles(); err != nil {
		t.Fatal(err)
	}

	// 验证文件1是否被删除
	if _, err := os.Stat(testFile1); !os.IsNotExist(err) {
		t.Error("file1 was not deleted")
	}

	// 验证文件2是否仍然存在
	if _, err := os.Stat(testFile2); os.IsNotExist(err) {
		t.Error("file2 was incorrectly deleted")
	}

	// 验证数据库状态
	var status string
	err = appDB.QueryRow("SELECT status FROM file_records WHERE source_path = ?", testFile1).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}
	if status != "deleted" {
		t.Error("file1 status was not updated to deleted")
	}

	err = appDB.QueryRow("SELECT status FROM file_records WHERE source_path = ?", testFile2).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}
	if status != "processed" {
		t.Error("file2 status was incorrectly updated")
	}
}
