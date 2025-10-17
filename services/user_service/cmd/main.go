package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/common/etcd"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/user_service/internal/config"
	"github.com/puoxiu/gogochat/services/user_service/internal/grpc_server"
	"github.com/puoxiu/gogochat/services/user_service/internal/http_server"
	user "github.com/puoxiu/gogochat/services/user_service/proto"
	"google.golang.org/grpc"
)

func main() {
	if err := config.InitConfig("/Users/xing/Desktop/test/go-ai/gogochat/services/user_service/etc/user.yaml"); err != nil {
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

	// 初始化 etcd 客户端 并注册服务
	etcdAddr := fmt.Sprintf("%s:%d", config.AppConfig.EtcdConfig.Host, config.AppConfig.EtcdConfig.Port)
	etcd.InitEtcd(etcdAddr)
	if err := etcd.Register(
		config.AppConfig.MainConfig.AppName,
		fmt.Sprintf("%s:%d", config.AppConfig.MainConfig.Host, config.AppConfig.MainConfig.GrpcPort),
	); err != nil {
		zlog.Fatal(fmt.Sprintf("注册服务到 etcd 失败: %v", err))
	}
	
	// 启动 gRPC 服务
	go func() {
		addr := fmt.Sprintf(":%d", config.AppConfig.MainConfig.GrpcPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			zlog.Fatal(fmt.Sprintf("启动 user gRPC 服务失败: %v", err))
		}

		s := grpc.NewServer()
		user.RegisterUserServiceServer(s, &grpc_server.UserGrpcServer{})
		zlog.Info(fmt.Sprintf("user gRPC 服务启动成功，端口：%s", addr))

		if err := s.Serve(lis); err != nil {
			zlog.Fatal(fmt.Sprintf("user gRPC 服务运行失败: %v", err))
		}
	}()

	// 启动 HTTP 服务
	go func() {
		addr := fmt.Sprintf("%s:%d", config.AppConfig.MainConfig.Host, config.AppConfig.MainConfig.HttpPort)
		zlog.Info(fmt.Sprintf("HTTP 服务启动成功，端口：%s", addr))
		if err := http_server.GE.Run(addr); err != nil {
			zlog.Fatal(fmt.Sprintf("HTTP 服务启动失败: %v", err))
		}
	}()

	// 初始化 session_service rpc 客户端--初始化失败时重试
	go func() {
		for {
			addr, err := etcd.GetServiceAddr("session_service")
			if err != nil {
				zlog.Error(fmt.Sprintf("获取 session_service 地址失败: %v", err))
				time.Sleep(time.Second * 10)
				continue
			}
			if len(addr) == 0 {
				zlog.Warn("未找到 session_service 服务，请先启动 session_service")
				time.Sleep(time.Second * 10)
				continue
			}

			if err := clients.InitGlobalSessionClient(addr[0]); err != nil {
				zlog.Warn(fmt.Sprintf("初始化 session_service rpc 客户端失败: %v", err))
				time.Sleep(time.Second * 10)
				continue
			}
			zlog.Info(fmt.Sprintf("成功初始化 session_service rpc 客户端: %s", addr[0]))
			break
		}
	}()


	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zlog.Info("收到退出信号，正在关闭 user_service ...")

	if sessionClient, err := clients.GetGlobalSessionClient(); err == nil {
		if closeErr := sessionClient.Close(); closeErr != nil {
			zlog.Warn(fmt.Sprintf("关闭session rpc客户端失败: %v", closeErr))
		} else {
			zlog.Info("session rpc客户端已关闭")
		}
	}
}
