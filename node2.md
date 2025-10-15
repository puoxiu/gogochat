# 微服务拆分记录

## 目标
user
session
chat



## 拆分user服务

### todo
1. 待实现的session服务接口：

// 会话服务接口（已定义，无需重复实现）
type SessionService interface {
    DeleteSessionsByUsers(sendId, receiveId string) error
}
