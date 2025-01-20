package processor

import (
	"database/sql"
	"fmt"
	"github.com/sleepstars/embypathrefresh/internal/model"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"time"
)

type Processor struct {
	embyDB     *sql.DB
	appDB      *sql.DB
	sourceDir  string
	targetDir  string
	logger     *zap.Logger
	deleteTime time.Duration
}

func New(embyDBPath, appDBPath, sourceDir, targetDir string, deleteTime time.Duration, logger *zap.Logger) (*Processor, error) {
	embyDB, err := sql.Open("sqlite3", embyDBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open emby database: %w", err)
	}

	appDB, err := sql.Open("sqlite3", appDBPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open app database: %w", err)
	}

	return &Processor{
		embyDB:     embyDB,
		appDB:      appDB,
		sourceDir:  sourceDir,
		targetDir:  targetDir,
		logger:     logger,
		deleteTime: deleteTime,
	}, nil
}

func (p *Processor) ProcessFile(record *model.FileRecord) error {
	// 检查文件是否已经处理过
	var exists bool
	err := p.appDB.QueryRow("SELECT EXISTS(SELECT 1 FROM file_records WHERE source_path = ? AND status != 'deleted')", 
		record.SourcePath).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check file existence: %w", err)
	}
	if exists {
		return nil
	}

	// 计算目标路径
	relPath, err := filepath.Rel(p.sourceDir, record.SourcePath)
	if err != nil {
		return fmt.Errorf("get relative path: %w", err)
	}
	record.TargetPath = filepath.Join(p.targetDir, relPath)

	// 在Emby数据库中更新路径
	tx, err := p.embyDB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 更新MediaItems表中的路径
	_, err = tx.Exec("UPDATE MediaItems SET Path = ? WHERE Path = ?", 
		record.TargetPath, record.SourcePath)
	if err != nil {
		return fmt.Errorf("update media items: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(record.TargetPath), 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	// 移动文件
	if err := os.Rename(record.SourcePath, record.TargetPath); err != nil {
		return fmt.Errorf("move file: %w", err)
	}

	// 记录处理状态
	now := time.Now()
	record.ProcessedTime = now
	if p.deleteTime > 0 {
		deleteTime := now.Add(p.deleteTime)
		record.DeleteScheduled = deleteTime
	}
	record.Status = "processed"
	record.CreatedAt = now
	record.UpdatedAt = now

	// 保存记录
	_, err = p.appDB.Exec(`
		INSERT INTO file_records (
			source_path, target_path, modified_time, processed_time, 
			delete_scheduled, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.SourcePath, record.TargetPath, record.ModifiedTime,
		record.ProcessedTime, record.DeleteScheduled, record.Status,
		record.CreatedAt, record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert record: %w", err)
	}

	return nil
}

func (p *Processor) CleanupFiles() error {
	rows, err := p.appDB.Query(`
		SELECT id, source_path FROM file_records 
		WHERE status = 'processed' 
		AND delete_scheduled IS NOT NULL 
		AND delete_scheduled <= ?`,
		time.Now())
	if err != nil {
		return fmt.Errorf("query files to delete: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var sourcePath string
		if err := rows.Scan(&id, &sourcePath); err != nil {
			p.logger.Error("scan row", zap.Error(err))
			continue
		}

		if err := os.Remove(sourcePath); err != nil {
			if !os.IsNotExist(err) {
				p.logger.Error("remove file", zap.Error(err), zap.String("path", sourcePath))
				continue
			}
		}

		_, err = p.appDB.Exec("UPDATE file_records SET status = 'deleted', updated_at = ? WHERE id = ?",
			time.Now(), id)
		if err != nil {
			p.logger.Error("update record status", zap.Error(err), zap.Int64("id", id))
		}
	}

	return rows.Err()
}

func (p *Processor) Close() error {
	if err := p.embyDB.Close(); err != nil {
		p.logger.Error("close emby database", zap.Error(err))
	}
	return p.appDB.Close()
}
