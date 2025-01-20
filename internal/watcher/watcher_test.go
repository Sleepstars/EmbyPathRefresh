package watcher

import (
	"github.com/sleepstars/embypathrefresh/internal/model"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type mockProcessor struct {
	processedFiles map[string]bool
	mu            sync.Mutex
	t             *testing.T
}

func newMockProcessor(t *testing.T) *mockProcessor {
	return &mockProcessor{
		processedFiles: make(map[string]bool),
		t:             t,
	}
}

func (m *mockProcessor) ProcessFile(record *model.FileRecord) error {
	m.mu.Lock()
	m.processedFiles[record.SourcePath] = true
	m.mu.Unlock()
	return nil
}

func (m *mockProcessor) waitForFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		processed := m.processedFiles[path]
		m.mu.Unlock()
		if processed {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func TestWatcher(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	processor := newMockProcessor(t)

	// 创建观察器
	w, err := New(tmpDir, 500*time.Millisecond, processor, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	// 启动观察器
	if err := w.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// 等待观察器初始化
	time.Sleep(200 * time.Millisecond)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 修改文件
	time.Sleep(600 * time.Millisecond) // 等待超过更新时间
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 等待处理完成
	if !processor.waitForFile(testFile, 2*time.Second) {
		t.Error("file was not processed")
	}

	// 测试子目录监控
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	// 在子目录中创建文件
	subFile := filepath.Join(subDir, "subtest.txt")
	if err := os.WriteFile(subFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(600 * time.Millisecond)
	if err := os.WriteFile(subFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 等待处理完成
	if !processor.waitForFile(subFile, 2*time.Second) {
		t.Error("subdirectory file was not processed")
	}
}
