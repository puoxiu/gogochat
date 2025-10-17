package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/services/session_service/internal/dao"
	"github.com/puoxiu/gogochat/services/session_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/session_service/internal/dto/respond"
	"github.com/puoxiu/gogochat/services/session_service/internal/model"

	"github.com/puoxiu/gogochat/pkg/constants"

	// "github.com/puoxiu/gogochat/pkg/enum/group_info/group_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/user_info/user_status_enum"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"gorm.io/gorm"
)

type sessionService struct {
}

var SessionService = new(sessionService)

// CreateSession 创建会话 
func (s *sessionService) CreateSession(req request.CreateSessionRequest) (string, string, int) {
	// 1. 获取全局用户客户端实例
	userClient, err := clients.GetGlobalUserClient()
	if err != nil {
		zlog.Error("获取用户客户端失败: " + err.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	var session model.Session
	session.Uuid = fmt.Sprintf("S%s", random.GetNowAndLenRandomString(11))
	session.SendId = req.SendId
	session.ReceiveId = req.ReceiveId
	session.CreatedAt = time.Now()
	if req.ReceiveId[0] == 'U' {
		receiveUserResp := userClient.GetUserInfo(req.ReceiveId)
		if receiveUserResp.Code == -1 {
			zlog.Error(fmt.Sprintf("获取用户信息RPC服务错误: uuid=%s, code=%d, msg=%s", req.ReceiveId, receiveUserResp.Code, receiveUserResp.Message))
			return constants.SYSTEM_ERROR, "", -1
		}
		if receiveUserResp.Code == -2 {
			zlog.Warn(fmt.Sprintf("获取用户信息业务错误: uuid=%s, code=%d, msg=%s", req.ReceiveId, receiveUserResp.Code, receiveUserResp.Message))
			return receiveUserResp.Message, "", -2
		}

		if receiveUserResp.Status == user_status_enum.DISABLE {
			zlog.Warn(fmt.Sprintf("该用户被禁用了: uuid=%s", req.ReceiveId))
			return "该用户被禁用了", "", -2
		} else {
			session.ReceiveName = receiveUserResp.Nickname
			session.Avatar = receiveUserResp.Avatar
		}
	} else {
		// todo group rpc
		// var receiveGroup model.GroupInfo
		// if res := dao.GormDB.Where("uuid = ?", req.ReceiveId).First(&receiveGroup); res.Error != nil {
		// 	zlog.Error(res.Error.Error())
		// 	return constants.SYSTEM_ERROR, "", -1
		// }
		// if receiveGroup.Status == group_status_enum.DISABLE {
		// 	zlog.Error("该群聊被禁用了")
		// 	return "该群聊被禁用了", "", -2
		// } else {
		// 	session.ReceiveName = receiveGroup.Name
		// 	session.Avatar = receiveGroup.Avatar
		// }
	}

	if res := dao.GormDB.Create(&session); res.Error != nil {
		zlog.Error(fmt.Sprintf("创建会话数据库错误: %s", res.Error.Error()))
		return constants.SYSTEM_ERROR, "", -1
	}
	if err := cache.GetGlobalCache().DelKeysWithPattern("group_session_list_" + req.SendId); err != nil {
		zlog.Warn(fmt.Sprintf("删除用户群聊会话缓存失败: %s", err.Error()))
	}
	if err := cache.GetGlobalCache().DelKeysWithPattern("session_list_" + req.SendId); err != nil {
		zlog.Warn(fmt.Sprintf("删除用户会话缓存失败: %s", err.Error()))
	}
	return "会话创建成功", session.Uuid, 0
}

// CheckOpenSessionAllowed 检查是否允许发起会话
// 目前的逻辑是：存在好友关系即可，拉黑也能发起会话 但是后续不能发送消息而已
func (s *sessionService) CheckOpenSessionAllowed(sendId, receiveId string) (string, bool, int) {
	userClient, err := clients.GetGlobalUserClient()
	if err != nil {
		zlog.Error("获取用户客户端失败: " + err.Error())
		return constants.SYSTEM_ERROR, false, -1
	}

	contactResp := userClient.GetUserContact(sendId, receiveId)
	// 处理RPC返回的业务code（按约定：code=-2无好友关系，code=0=有好友关系，code=-1系统错误）
	if contactResp.Code == -1 {
		zlog.Error(fmt.Sprintf("查询好友关系RPC服务错误: sendId=%s, receiveId=%s, code=%d, msg=%s", sendId, receiveId, contactResp.Code, contactResp.Message))
		return constants.SYSTEM_ERROR, false, -1
	}
	if contactResp.Code == -2 {
		return "不是好友关系，无法发起会话", false, 0
	}
	// // 存在好友关系：校验拉黑状态
	// if contactResp.Contact.Status == contact_status_enum.BE_BLACK {
	// 	return "已被对方拉黑，无法发起会话", false, 0
	// }
	// if contactResp.Contact.Status == contact_status_enum.BLACK {
	// 	return "已拉黑对方，先解除拉黑状态才能发起会话", false, 0
	// }

	if receiveId[0] == 'U' {
		userStatusResp := userClient.GetUserInfo(receiveId)
		if userStatusResp.Code == -1 {
			zlog.Error(fmt.Sprintf("查询接收者用户状态RPC服务错误: sendId=%s, receiveId=%s, code=%d, msg=%s", sendId, receiveId, userStatusResp.Code, userStatusResp.Message))
			return constants.SYSTEM_ERROR, false, -1
		}
		// 此时不可能为0了
		if userStatusResp.Code == -2 {
			zlog.Warn(fmt.Sprintf("获取用户信息业务失败: uuid=%s, code=%d, msg=%s", receiveId, userStatusResp.Code, userStatusResp.Message))
			return userStatusResp.Message, false, -2
		}

		// if userStatusResp.Status == user_status_enum.DISABLE {
		// 	zlog.Info(fmt.Sprintf("该用户被禁用了: uuid=%s", receiveId))
		// 	return "对方已被禁用，无法发起会话", false, -2
		// }
	} else {
		// var group model.GroupInfo
		// if res := dao.GormDB.Where("uuid = ?", receiveId).First(&group); res.Error != nil {
		// 	zlog.Error(res.Error.Error())
		// 	return constants.SYSTEM_ERROR, false, -1
		// }
		// if group.Status == group_status_enum.DISABLE {
		// 	zlog.Info("对方已被禁用，无法发起会话")
		// 	return "对方已被禁用，无法发起会话", false, -2
		// }
	}
	return "可以发起会话", true, 0
}

// DeleteSession 删除会话

// OpenSession 打开会话 -✅
func (s *sessionService) OpenSession(req request.OpenSessionRequest) (string, string, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("session_" + req.SendId + "_" + req.ReceiveId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var session model.Session
			if res := dao.GormDB.Where("send_id = ? and receive_id = ?", req.SendId, req.ReceiveId).First(&session); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Info("会话没有找到，将新建会话")
					createReq := request.CreateSessionRequest{
						SendId:    req.SendId,
						ReceiveId: req.ReceiveId,
					}
					return s.CreateSession(createReq)
				}
				zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", res.Error.Error()))
				return constants.SYSTEM_ERROR, "", -1
			}
			// 数据库存在，而缓存不存在，将数据库会话缓存到Redis
			rspString, err := json.Marshal(session)
			if err != nil {
				zlog.Error(fmt.Sprintf("会话序列化错误: %s", err.Error()))
				return constants.SYSTEM_ERROR, "", -1
			}
			if err := cache.GetGlobalCache().SetKeyEx("session_"+req.SendId+"_"+req.ReceiveId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Warn(fmt.Sprintf("缓存会话错误: %s", err.Error()))
			}
			return "打开会话成功", session.Uuid, 0
		} else {
			zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", err.Error()))
			return constants.SYSTEM_ERROR, "", -1
		}
	}
	var session model.Session
	fmt.Println("rspString:====", rspString)
	if err := json.Unmarshal([]byte(rspString), &session); err != nil {
		zlog.Error(fmt.Sprintf("会话反序列化错误: %s", err.Error()))
		return constants.SYSTEM_ERROR, "", -1
	}
	return "打开会话成功", session.Uuid, 0
}

// GetUserSessionList 获取用户会话列表 -✅
func (s *sessionService) GetUserSessionList(ownerId string) (string, []respond.UserSessionListRespond, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("session_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var sessionList []model.Session
			if res := dao.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Info(fmt.Sprintf("用户 %s 未创建会话", ownerId))
					return "未创建用户会话", nil, -2
				} else {
					zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", res.Error.Error()))
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var sessionListRsp []respond.UserSessionListRespond
			for i := 0; i < len(sessionList); i++ {
				if sessionList[i].ReceiveId[0] == 'U' {
					sessionListRsp = append(sessionListRsp, respond.UserSessionListRespond{
						SessionId: sessionList[i].Uuid,
						Avatar:    sessionList[i].Avatar,
						UserId:    sessionList[i].ReceiveId,
						Username:  sessionList[i].ReceiveName,
					})
				}
			}
			rspString, err := json.Marshal(sessionListRsp)
			if err != nil {
				zlog.Error(fmt.Sprintf("会话序列化错误: %s", err.Error()))
				return constants.SYSTEM_ERROR, nil, -1
			}
			if err := cache.GetGlobalCache().SetKeyEx("session_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Warn(fmt.Sprintf("缓存会话列表错误: %s", err.Error()))
			}
			return "获取成功", sessionListRsp, 0
		} else {
			zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", err.Error()))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.UserSessionListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(fmt.Sprintf("会话反序列化错误: %s", err.Error()))
		return constants.SYSTEM_ERROR, nil, -1
	}
	return "获取成功", rsp, 0
}

// GetGroupSessionList 获取群聊会话列表 -✅
func (s *sessionService) GetGroupSessionList(ownerId string) (string, []respond.GroupSessionListRespond, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("group_session_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var sessionList []model.Session
			if res := dao.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Info(fmt.Sprintf("用户 %s 未创建群聊会话", ownerId))
					return "未创建群聊会话", nil, -2
				} else {
					zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", res.Error.Error()))
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var sessionListRsp []respond.GroupSessionListRespond
			for i := 0; i < len(sessionList); i++ {
				if sessionList[i].ReceiveId[0] == 'G' {
					sessionListRsp = append(sessionListRsp, respond.GroupSessionListRespond{
						SessionId: sessionList[i].Uuid,
						Avatar:    sessionList[i].Avatar,
						GroupId:   sessionList[i].ReceiveId,
						GroupName: sessionList[i].ReceiveName,
					})
				}
			}
			rspString, err := json.Marshal(sessionListRsp)
			if err != nil {
				zlog.Error(err.Error())
			}
			if err := cache.GetGlobalCache().SetKeyEx("group_session_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Warn(fmt.Sprintf("缓存群聊会话列表错误: %s", err.Error()))
			}
			return "获取成功", sessionListRsp, 1
		} else {
			zlog.Error(fmt.Sprintf("查询会话数据库错误: %s", err.Error()))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.GroupSessionListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(fmt.Sprintf("会话反序列化错误: %s", err.Error()))
		return constants.SYSTEM_ERROR, nil, -1
	}
	return "获取成功", rsp, 1
}

// DeleteSession 删除会话 -✅
func (s *sessionService) DeleteSession(ownerId, sessionId string) (string, int) {
	var session model.Session
	if res := dao.GormDB.Where("uuid = ?", sessionId).First(&session); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info(fmt.Sprintf("删除会话: %s 不存在", sessionId))
			return "该会话不存在", -2
		} else {
			zlog.Error(fmt.Sprintf("删除会话数据库错误: %s", res.Error.Error()))
			return constants.SYSTEM_ERROR, -1
		}
	}
	session.DeletedAt.Valid = true
	session.DeletedAt.Time = time.Now()
	if res := dao.GormDB.Save(&session); res.Error != nil {
		zlog.Error(fmt.Sprintf("删除会话数据库错误: %s", res.Error.Error()))
		return constants.SYSTEM_ERROR, -1
	}
	if err := cache.GetGlobalCache().DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
		zlog.Warn(fmt.Sprintf("删除缓存群聊会话列表失败: %s", err.Error()))
	}
	if err := cache.GetGlobalCache().DelKeysWithPattern("session_list_" + ownerId); err != nil {
		zlog.Warn(fmt.Sprintf("删除缓存会话列表失败: %s", err.Error()))
	}
	return "删除成功", 1
}
