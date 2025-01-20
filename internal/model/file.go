package model

import "time"

// FileRecord 记录文件迁移状态
type FileRecord struct {
	ID              int64     `db:"id"`
	SourcePath      string    `db:"source_path"`
	TargetPath      string    `db:"target_path"`
	ModifiedTime    time.Time `db:"modified_time"`
	ProcessedTime   time.Time `db:"processed_time"`
	DeleteScheduled time.Time `db:"delete_scheduled"`
	Status          string    `db:"status"` // pending, processed, deleted
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}
