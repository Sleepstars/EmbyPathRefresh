package config

import (
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	App struct {
		Name    string
		Version string
	}
	Paths struct {
		SourceDir string `mapstructure:"source_dir"`
		TargetDir string `mapstructure:"target_dir"`
		EmbyDB    string `mapstructure:"emby_db"`
	}
	Timings struct {
		UpdateAfter time.Duration `mapstructure:"update_after"`
		DeleteAfter time.Duration `mapstructure:"delete_after"`
	}
	Database struct {
		Path string
	}
	Logging struct {
		Level string
		File  string
	}
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 将小时转换为持续时间
	config.Timings.UpdateAfter *= time.Hour
	config.Timings.DeleteAfter *= time.Hour

	return &config, nil
}
