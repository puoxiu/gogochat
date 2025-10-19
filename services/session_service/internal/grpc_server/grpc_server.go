package grpc_server

import (
	"context"

	"github.com/puoxiu/gogochat/services/session_service/internal/services"
	session "github.com/puoxiu/gogochat/services/session_service/proto"
)


type SessionGrpcServer struct {
	session.UnimplementedSessionServiceServer
}


func (s *SessionGrpcServer) DeleteSessionsByUsers( ctx context.Context, req *session.DeleteSessionsByUsersRequest) (*session.DeleteSessionsByUsersResponse, error) {
	if req.SendId == "" || req.ReceiveId == "" {
		return &session.DeleteSessionsByUsersResponse{
			Code:    -1,
			Message: "参数错误：发送者或接收者ID不能为空",
		}, nil
	}

	msg, code := services.SessionService.DeleteSessionBySendIdAndReceiveId(req.SendId, req.ReceiveId)

	return &session.DeleteSessionsByUsersResponse{
		Code:    int32(code),
		Message: msg,
	}, nil
}

func (s *SessionGrpcServer) CreateSessionIfNotExist( ctx context.Context, req *session.CreateSessionRequest) (*session.CreateSessionResponse, error) {
	if req.SendId == "" || req.ReceiveId == "" {
		return &session.CreateSessionResponse{
			Code:    -1,
			Message: "参数错误：发送者或接收者ID不能为空",
		}, nil
	}

	msg, _, code := services.SessionService.OpenSession(req.SendId, req.ReceiveId)

	return &session.CreateSessionResponse{
		Code:    int32(code),
		Message: msg,
	}, nil
}
