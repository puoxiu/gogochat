package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/puoxiu/gogochat/config"
	"github.com/puoxiu/gogochat/internal/https_server"
	"github.com/puoxiu/gogochat/internal/service/chat"
	"github.com/puoxiu/gogochat/internal/service/kafka"
	myredis "github.com/puoxiu/gogochat/internal/service/redis"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

func main() {
	// 1. 配置文件类初始化
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	kafkaConfig := conf.KafkaConfig

	if kafkaConfig.MessageMode == "channel" {
		go chat.ChatServer.Start()
	} else {
		kafka.KafkaService.KafkaInit()
		go chat.KafkaChatServer.Start()
	}

	go func() {
		// 使用TLS协议 加密通信
		// if err := https_server.GE.RunTLS(fmt.Sprintf("%s:%d", host, port), "/etc/ssl/certs/server.crt", "/etc/ssl/private/server.key"); err != nil {
		// 	zlog.Fatal("server running fault")
		// 	return
		// }

		// 开发阶段 未加密通信
		fmt.Printf("development server running on %s:%d\n", host, port)
		if err := https_server.GE.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			zlog.Fatal("server running fault")
			return
		}
	}()

	// 设置信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	// 回收资源
	if kafkaConfig.MessageMode == "kafka" {
		kafka.KafkaService.KafkaClose()
	}
	chat.ChatServer.Close()
	zlog.Info("server closing...")

	// 删除所有的redis键
	if err := myredis.DeleteAllRedisKeys(); err != nil {
		zlog.Error(err.Error())
	} else {
		zlog.Info("所有Redis键已删除")
	}
	zlog.Info("server closed and redis keys deleted")
}
