package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/session_service/internal/config"
	"github.com/puoxiu/gogochat/services/session_service/internal/http_server"
)

func main() {
	if err := config.InitConfig("/Users/xing/Desktop/test/go-ai/gogochat/services/session_service/etc/session.yaml"); err != nil {
		zlog.Fatal(fmt.Sprintf("初始化配置失败: %v", err))
	}

	// 初始化缓存
	redisCache := cache.NewRedisCache(
		context.Background(),
		config.AppConfig.RedisConfig.Host,
		config.AppConfig.RedisConfig.Port,
		config.AppConfig.RedisConfig.Password,
		config.AppConfig.RedisConfig.DB,
	)
	if redisCache == nil {
		zlog.Fatal("初始化 Redis 缓存失败")
	}
	cache.Init(redisCache)

	// 连接RPC服务-地址先硬编码 之后可以用etcd等服务发现
	userGrpcAddr := fmt.Sprintf("%s:%d", "127.0.0.1", 9001)
	if err := clients.InitGlobalUserClient(userGrpcAddr); err != nil {
		zlog.Fatal(fmt.Sprintf("初始化user rpc客户端失败: %v", err))
	}

	// 启动seession gRPC 服务

	// 启动 HTTP 服务
	go func() {
		addr := fmt.Sprintf("%s:%d", config.AppConfig.MainConfig.Host, config.AppConfig.MainConfig.HttpPort)
		zlog.Info(fmt.Sprintf("HTTP 服务启动成功，端口：%s", addr))
		if err := http_server.GE.Run(addr); err != nil {
			zlog.Fatal(fmt.Sprintf("HTTP 服务启动失败: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zlog.Info("收到退出信号，正在关闭 user_service ...")

	if userClient, err := clients.GetGlobalUserClient(); err == nil {
		if closeErr := userClient.Close(); closeErr != nil {
			zlog.Warn(fmt.Sprintf("关闭user rpc客户端失败: %v", closeErr))
		} else {
			zlog.Info("user rpc客户端已关闭")
		}
	}
}
