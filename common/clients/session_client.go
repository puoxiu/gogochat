package clients

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
	sessionpb "github.com/puoxiu/gogochat/services/session_service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionClient struct {
	client sessionpb.SessionServiceClient
	conn   *grpc.ClientConn
}

var (
	globalSessionClient atomic.Value
	onceSessionInit     sync.Once
)

func NewSessionClient(addr string) (*SessionClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("连接会话服务失败: %v", err)
	}

	client := sessionpb.NewSessionServiceClient(conn)
	return &SessionClient{client: client, conn: conn}, nil
}

// 改造点3️⃣：原 once.Do 逻辑保留，但使用 atomic.Store
func InitGlobalSessionClient(addr string) error {
	var err error
	onceSessionInit.Do(func() {
		client, initErr := NewSessionClient(addr)
		if initErr != nil {
			err = initErr
			zlog.Error(fmt.Sprintf("全局会话客户端初始化失败: %v", err))
			return
		}
		globalSessionClient.Store(client)
		zlog.Info("全局会话客户端初始化成功")
	})
	return err
}

// 改造点4️⃣：统一通过 atomic.Load 获取全局客户端
func GetGlobalSessionClient() (*SessionClient, error) {
	client, ok := globalSessionClient.Load().(*SessionClient)
	if !ok || client == nil {
		return nil, fmt.Errorf("会话客户端未初始化，请先调用 InitGlobalSessionClient")
	}
	return client, nil
}

// DeleteSessionsByUsers 删除会话
func (sc *SessionClient) DeleteSessionsByUsers(userId, contactId string) *sessionpb.DeleteSessionsByUsersResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := sc.client.DeleteSessionsByUsers(ctx, &sessionpb.DeleteSessionsByUsersRequest{
		SendId:    userId,
		ReceiveId: contactId,
	})
	return resp
}

// CreateSessionIfNotExist 创建会话（若不存在）
func (sc *SessionClient) CreateSessionIfNotExist(sendId, receiveId string) *sessionpb.CreateSessionResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, _ := sc.client.CreateSessionIfNotExist(ctx, &sessionpb.CreateSessionRequest{
		SendId:    sendId,
		ReceiveId: receiveId,
	})
	return resp
}

// Close 关闭连接
func (sc *SessionClient) Close() error {
	if sc.conn != nil {
		return sc.conn.Close()
	}
	return nil
}
