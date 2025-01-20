package main

import (
	"flag"
	"github.com/sleepstars/embypathrefresh/internal/config"
	"github.com/sleepstars/embypathrefresh/internal/processor"
	"github.com/sleepstars/embypathrefresh/internal/watcher"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	// 初始化日志
	logConfig := zap.NewProductionConfig()
	logConfig.OutputPaths = []string{cfg.Logging.File}
	logConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if cfg.Logging.Level == "debug" {
		logConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	
	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(cfg.Logging.File), 0755); err != nil {
		logger.Fatal("create log directory failed", zap.Error(err))
	}

	// 初始化处理器
	proc, err := processor.New(
		cfg.Paths.EmbyDB,
		cfg.Database.Path,
		cfg.Paths.SourceDir,
		cfg.Paths.TargetDir,
		cfg.Timings.DeleteAfter,
		logger,
	)
	if err != nil {
		logger.Fatal("create processor failed", zap.Error(err))
	}
	defer proc.Close()

	// 初始化文件监控
	w, err := watcher.New(
		cfg.Paths.SourceDir,
		cfg.Timings.UpdateAfter,
		proc,
		logger,
	)
	if err != nil {
		logger.Fatal("create watcher failed", zap.Error(err))
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		logger.Fatal("start watcher failed", zap.Error(err))
	}

	// 定期清理文件
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			if err := proc.CleanupFiles(); err != nil {
				logger.Error("cleanup files failed", zap.Error(err))
			}
		}
	}()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down...")
	ticker.Stop()
}
