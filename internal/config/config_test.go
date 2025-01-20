package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
app:
  name: TestApp
  version: 1.0.0
paths:
  source_dir: /test/source
  target_dir: /test/target
  emby_db: /test/library.db
timings:
  update_after: 24
  delete_after: 168
database:
  path: ./data/test.db
logging:
  level: debug
  file: ./logs/test.log
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 测试加载配置
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证配置值
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"app.name", cfg.App.Name, "TestApp"},
		{"app.version", cfg.App.Version, "1.0.0"},
		{"paths.source_dir", cfg.Paths.SourceDir, "/test/source"},
		{"paths.target_dir", cfg.Paths.TargetDir, "/test/target"},
		{"paths.emby_db", cfg.Paths.EmbyDB, "/test/library.db"},
		{"timings.update_after", cfg.Timings.UpdateAfter, 24 * time.Hour},
		{"timings.delete_after", cfg.Timings.DeleteAfter, 168 * time.Hour},
		{"database.path", cfg.Database.Path, "./data/test.db"},
		{"logging.level", cfg.Logging.Level, "debug"},
		{"logging.file", cfg.Logging.File, "./logs/test.log"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}

	// 测试错误情况
	t.Run("non-existent file", func(t *testing.T) {
		_, err := Load("non-existent.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}
