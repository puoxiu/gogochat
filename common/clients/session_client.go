package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
	sessionpb "github.com/puoxiu/gogochat/services/session_service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionClient struct {
	client sessionpb.SessionServiceClient // GRPC生成的客户端接口
	conn   *grpc.ClientConn         // GRPC连接实例
}

var (
	globalSessionClient *SessionClient
)

// NewSessionClient 创建会话服务RPC客户端
// addr: 会话服务的GRPC地址（格式："ip:port"，如"session-service:50051"）
func NewSessionClient(addr string) (*SessionClient, error) {
	// 建立GRPC连接（生产环境建议添加TLS配置）
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), 
		grpc.WithTimeout(5*time.Second),                          
	)
	if err != nil {
		return nil, fmt.Errorf("连接会话服务失败: %v", err)
	}

	client := sessionpb.NewSessionServiceClient(conn)
	return &SessionClient{
		client: client,
		conn:   conn,
	}, nil
}

// InitGlobalSessionClient 初始化全局会话服务客户端（程序启动时调用一次）
func InitGlobalSessionClient(addr string) error {
	var err error
	once.Do(func() {
		globalSessionClient, err = NewSessionClient(addr)
		if err != nil {
			zlog.Error(fmt.Sprintf("全局会话客户端初始化失败: %v", err))
		} else {
			zlog.Info("全局会话客户端初始化成功")
		}
	})
	return err
}

// GetGlobalSessionClient 获取全局会话服务客户端实例
func GetGlobalSessionClient() (*SessionClient, error) {
	if globalSessionClient == nil {
		return nil, fmt.Errorf("会话客户端未初始化，请先调用InitGlobalSessionClient")
	}
	return globalSessionClient, nil
}

// DeleteSessionsByUsers 调用会话服务删除用户会话
func (sc *SessionClient) DeleteSessionsByUsers(userId string, contactId string) (*sessionpb.DeleteSessionsByUsersResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := sc.client.DeleteSessionsByUsers(ctx, &sessionpb.DeleteSessionsByUsersRequest{
		SendId:    userId,
		ReceiveId: contactId,
	})

	return resp
}


// Close 关闭GRPC连接，释放资源
// 在程序退出时调用（如main函数的defer中）
func (sc *SessionClient) Close() error {
	if sc.conn != nil {
		return sc.conn.Close()
	}
	return nil
}