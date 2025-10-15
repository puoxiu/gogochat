package grpc_server

import (
	"context"

	"github.com/puoxiu/gogochat/services/user_service/internal/services"
	user "github.com/puoxiu/gogochat/services/user_service/proto"
)

type UserGrpcServer struct {
	user.UnimplementedUserServiceServer // 必须嵌入
}

func (s *UserGrpcServer) GetUserInfo(ctx context.Context, req *user.GetUserInfoRequest) (*user.GetUserInfoResponse, error) {
	msg, rsp, code := services.UserInfoService.GetUserInfo(req.Uuid)
	if code != 0 {
		return &user.GetUserInfoResponse{
			Code:    int32(code),
			Message: msg,
		}, nil
	}
	return &user.GetUserInfoResponse{
		Uuid:      rsp.Uuid,
		Nickname:  rsp.Nickname,
		Telephone: rsp.Telephone,
		Avatar:    rsp.Avatar,
		Email:     rsp.Email,
		Gender:    int32(rsp.Gender),
		Birthday:  rsp.Birthday,
		Signature: rsp.Signature,
		CreatedAt: rsp.CreatedAt,
		IsAdmin:   int32(rsp.IsAdmin),
		Status:    int32(rsp.Status),
		Message:   msg,
		Code:      0,
	}, nil
}
