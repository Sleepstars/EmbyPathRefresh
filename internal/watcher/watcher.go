package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sleepstars/embypathrefresh/internal/model"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileProcessor interface {
	ProcessFile(record *model.FileRecord) error
}

type Watcher struct {
	watcher    *fsnotify.Watcher
	processor  FileProcessor
	sourceDir  string
	logger     *zap.Logger
	updateTime time.Duration
	mu         sync.Mutex
	// 添加一个map来跟踪最后修改时间
	lastModified map[string]time.Time
}

func New(sourceDir string, updateTime time.Duration, processor FileProcessor, logger *zap.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:      fsWatcher,
		processor:    processor,
		sourceDir:    sourceDir,
		logger:       logger,
		updateTime:   updateTime,
		lastModified: make(map[string]time.Time),
	}

	return w, nil
}

func (w *Watcher) Start() error {
	// 递归添加所有子目录
	if err := filepath.Walk(w.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	}); err != nil {
		return err
	}

	go w.watchLoop()
	return nil
}

func (w *Watcher) watchLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.handleFileModification(event.Name)
			}
			
			// 如果有新目录创建，添加到监控列表
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", zap.Error(err))
		}
	}
}

func (w *Watcher) handleFileModification(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		w.logger.Error("stat file error", zap.Error(err), zap.String("path", path))
		return
	}

	lastMod, exists := w.lastModified[path]
	if exists && time.Since(lastMod) < w.updateTime {
		return
	}

	w.lastModified[path] = time.Now()

	record := &model.FileRecord{
		SourcePath:   path,
		ModifiedTime: info.ModTime(),
		Status:      "pending",
	}

	if err := w.processor.ProcessFile(record); err != nil {
		w.logger.Error("process file error", zap.Error(err), zap.String("path", path))
	}
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}
