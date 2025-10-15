package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
	userpb "github.com/puoxiu/gogochat/services/user_service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	client userpb.UserServiceClient // GRPC生成的客户端接口
	conn   *grpc.ClientConn         // GRPC连接实例
}

var (
	globalUserClient *UserClient
	once             sync.Once
)

// NewUserClient 创建用户服务RPC客户端
// addr: 用户服务的GRPC地址（格式："ip:port"，如"user-service:50051"）
func NewUserClient(addr string) (*UserClient, error) {
	// 建立GRPC连接（生产环境建议添加TLS配置）
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), 
		grpc.WithTimeout(5*time.Second),                          
	)
	if err != nil {
		return nil, fmt.Errorf("连接用户服务失败: %v", err)
	}

	client := userpb.NewUserServiceClient(conn)
	return &UserClient{
		client: client,
		conn:   conn,
	}, nil
}

// InitGlobalUserClient 初始化全局用户服务客户端（程序启动时调用一次）
func InitGlobalUserClient(addr string) error {
	var err error
	once.Do(func() {
		globalUserClient, err = NewUserClient(addr)
		if err != nil {
			zlog.Error(fmt.Sprintf("全局用户客户端初始化失败: %v", err))
		} else {
			zlog.Info("全局用户客户端初始化成功")
		}
	})
	return err
}

// GetGlobalUserClient 获取全局用户服务客户端实例
func GetGlobalUserClient() (*UserClient, error) {
	if globalUserClient == nil {
		return nil, fmt.Errorf("用户客户端未初始化，请先调用InitGlobalUserClient")
	}
	return globalUserClient, nil
}

// GetUserInfo 调用用户服务查询用户信息
func (uc *UserClient) GetUserInfo(uuid string) (*userpb.GetUserInfoResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := uc.client.GetUserInfo(ctx, &userpb.GetUserInfoRequest{Uuid: uuid})
	return resp
}

// GetUserContact 调用用户服务查询用户好友关系
func (uc *UserClient) GetUserContact(userId string, contactId string) (*userpb.GetUserContactResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := uc.client.GetUserContact(ctx, &userpb.GetUserContactRequest{
		UserId:    userId,
		ContactId: contactId,
	})

	return resp
}

// Close 关闭GRPC连接，释放资源
// 在程序退出时调用（如main函数的defer中）
func (uc *UserClient) Close() error {
	if uc.conn != nil {
		return uc.conn.Close()
	}
	return nil
}