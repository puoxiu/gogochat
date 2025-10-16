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
	if code != 1 {
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
		Code:      1,
	}, nil
}

func (s *UserGrpcServer) GetUserContact(ctx context.Context, req *user.GetUserContactRequest) (*user.GetUserContactResponse, error) {
	msg, rsp, code := services.UserContactService.GetUserContact(req.UserId, req.ContactId)

	// code = 0 表示查询成功; code = -1 表示查询失败; code = -2 查询正常 但是数据不存在(业务错误)
	if code != 0 {
		return &user.GetUserContactResponse{
			Code:    int32(code),
			Message: msg,
			Contact: nil,
		}, nil
	}
	return &user.GetUserContactResponse{
		Code:      0,
		Message:   msg,
		Contact: &user.FriendContact{
			UserId:     rsp.UserId,
			ContactId:  rsp.ContactId,
			Status:     int32(rsp.Status),
			ContactType: int32(rsp.ContactType),
		},
	}, nil
}