# Emby Path Refresh

自动化管理Emby媒体文件路径的工具。监控指定目录的文件变化，自动更新Emby数据库中的文件路径，并支持文件迁移和清理功能。

## 功能特性

- 🔍 实时监控指定目录的文件变化
- 🔄 自动更新Emby数据库中的文件路径
- 📦 支持文件迁移到新位置
- ⏰ 可配置的文件处理延迟时间
- 🗑️ 可选的源文件自动清理功能
- 📝 完整的操作日志记录

## 系统要求

- Go 1.21+
- SQLite 3.x
- Windows/Linux

## 快速开始

### 1. 获取代码

```bash
git clone https://github.com/sleepstars/embypathrefresh.git
cd embypathrefresh
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 编译程序

```bash
go build -o embypathrefresh ./cmd/embypathrefresh
```

### 4. 配置文件

复制并修改配置文件：

```yaml
app:
  name: EmbyPathRefresh
  version: 1.0.0

paths:
  source_dir: /path/to/source    # 源文件目录
  target_dir: /path/to/target    # 目标文件目录
  emby_db: /path/to/library.db   # Emby数据库路径

timings:
  update_after: 24   # 文件修改后等待时间（小时）
  delete_after: 168  # 文件删除等待时间（小时）

database:
  path: ./data/app.db

logging:
  level: info
  file: ./logs/app.log
```

### 5. 运行程序

```bash
./embypathrefresh.exe -config config.yaml
```

## 许可证

MIT License