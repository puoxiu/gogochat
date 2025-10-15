package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// 整体配置入口
type Config struct {
	MainConfig      MainConfig      `mapstructure:"main_config"`
	MySQLConfig     MySQLConfig     `mapstructure:"mysql_config"`
	RedisConfig     RedisConfig     `mapstructure:"redis_config"`
	LogConfig       LogConfig       `mapstructure:"log_config"`
}

// 服务基础配置
type MainConfig struct {
	AppName  string `mapstructure:"app_name"`
	Host     string `mapstructure:"host"`
	GrpcPort int    `mapstructure:"grpc_port"`
	HttpPort int    `mapstructure:"http_port"`
}

// MySQL配置
type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"database_name"`
}

// Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// 日志配置
type LogConfig struct {
	LogPath string `mapstructure:"log_path"`
}

var AppConfig *Config


// 初始化配置
func InitConfig(configPath string) error {
	v := viper.New()

	v.SetConfigFile(configPath)   // 指定配置文件路径
	v.SetConfigType("yaml")       // 指定文件类型

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	AppConfig = &cfg
	return nil
}

