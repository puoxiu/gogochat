# 微服务拆分记录

## 目标
user
session
chat



## 拆分user服务

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative services/user_service/proto/user.proto

### todo
1. 待实现的session服务接口：

// 会话服务接口（已定义，无需重复实现）
type SessionService interface {
    DeleteSessionsByUsers(sendId, receiveId string) error
}

## 拆分session服务

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative services/session_service/proto/session.proto

### todo
1. 待实现的group服务接口：
获取群组信息

