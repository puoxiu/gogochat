package clients

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
	userpb "github.com/puoxiu/gogochat/services/user_service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

// 关键修改：用atomic.Value替代普通变量，保证可见性
var (
	globalUserClient atomic.Value 
	once             sync.Once
)

// NewUserClient 逻辑不变
func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("连接用户服务失败: %v", err)
	}
	client := userpb.NewUserServiceClient(conn)
	return &UserClient{client: client, conn: conn}, nil
}

// InitGlobalUserClient 初始化逻辑修改：用Store存储客户端
func InitGlobalUserClient(addr string) error {
	var err error
	once.Do(func() {
		client, initErr := NewUserClient(addr)
		if initErr != nil {
			err = initErr
			zlog.Error(fmt.Sprintf("全局用户客户端初始化失败: %v", err))
			return
		}
		// 用atomic.Store存储客户端实例，保证可见性
		globalUserClient.Store(client)
		zlog.Info("全局用户客户端初始化成功")
	})
	return err
}

// GetGlobalUserClient 获取逻辑修改：用Load读取客户端
func GetGlobalUserClient() (*UserClient, error) {
	// 用atomic.Load读取最新值
	client, ok := globalUserClient.Load().(*UserClient)
	if !ok || client == nil {
		return nil, fmt.Errorf("用户客户端未初始化，请先调用InitGlobalUserClient")
	}
	return client, nil
}

// 其余方法（GetUserInfo、Close等）逻辑不变

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

// GetContactStatus 调用用户服务查询联系人状态
func (uc *UserClient) GetContactStatus(userId string, contactId string) (*userpb.GetContactStatusResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := uc.client.GetContactStatus(ctx, &userpb.GetContactStatusRequest{
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