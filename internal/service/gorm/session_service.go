package gorm

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"github.com/puoxiu/gogochat/internal/dao"
	"github.com/puoxiu/gogochat/internal/dto/request"
	"github.com/puoxiu/gogochat/internal/dto/respond"
	"github.com/puoxiu/gogochat/internal/model"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"time"
)

type sessionService struct {
}

var SessionService = new(sessionService)

// CreateSession 创建会话
func (s *sessionService) CreateSession(req request.CreateSessionRequest) (string, error) {
	var user model.UserInfo
	if res := dao.GormDB.Where("uuid = ?", req.SendId).First(&user); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Error(" send user not found")
			return "", errors.New("user not found")
		} else {
			zlog.Error(res.Error.Error())
			return "", res.Error
		}
	}
	if user.DeletedAt.Valid {
		zlog.Info("user has been deleted")
		return "", nil
	}
	if req.ReceiveId[0] == 'U' {
		var receiveUser model.UserInfo
		if res := dao.GormDB.Where("uuid = ?", req.ReceiveId).First(&receiveUser); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("receive user not found")
				return "", errors.New("receive user not found")
			} else {
				zlog.Error(res.Error.Error())
				return "", res.Error
			}
		}
		if receiveUser.DeletedAt.Valid {
			zlog.Info("receive user has been deleted")
			return "", nil
		}
	} else {
		var receiveGroup model.GroupInfo
		if res := dao.GormDB.Where("uuid = ?", req.ReceiveId).First(&receiveGroup); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("receive group not found")
				return "", errors.New("receive group not found")
			} else {
				zlog.Error(res.Error.Error())
				return "", res.Error
			}
		}
		if receiveGroup.DeletedAt.Valid {
			zlog.Info("receive group has been deleted")
			return "", nil
		}
	}
	var session model.Session
	session.Uuid = fmt.Sprintf("S%s", random.GetNowAndLenRandomString(11))
	session.SendId = req.SendId
	session.ReceiveId = req.ReceiveId
	session.CreatedAt = time.Now()
	if req.ReceiveId[0] == 'U' {
		var user model.UserInfo
		if res := dao.GormDB.Where("uuid = ?", req.ReceiveId).First(&user); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("receive user not found")
				return "", errors.New("receive user not found")
			} else {
				zlog.Error(res.Error.Error())
				return "", res.Error
			}
		}
		if !user.DeletedAt.Valid {
			session.ReceiveName = user.Nickname
			session.Avatar = user.Avatar
		}
	} else {
		var group model.GroupInfo
		if res := dao.GormDB.Where("uuid = ?", req.ReceiveId).First(&group); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("receive group not found")
				return "", errors.New("receive group not found")
			} else {
				zlog.Error(res.Error.Error())
				return "", res.Error
			}
		}
		if !group.DeletedAt.Valid {
			session.ReceiveName = group.Name
			session.Avatar = group.Avatar
		}
	}
	if res := dao.GormDB.Create(&session); res.Error != nil {
		zlog.Error(res.Error.Error())
		return "", res.Error
	}
	return session.Uuid, nil
}

// OpenSession 打开会话
func (s *sessionService) OpenSession(req request.OpenSessionRequest) (string, error) {
	var session model.Session
	if res := dao.GormDB.Where("send_id = ? and receive_id = ?", req.SendId, req.ReceiveId).First(&session); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("session not found")
			createReq := request.CreateSessionRequest{
				SendId:    req.SendId,
				ReceiveId: req.ReceiveId,
			}
			var uuid string
			var err error
			if uuid, err = s.CreateSession(createReq); err != nil {
				return "", err
			}
			return uuid, nil
		}
	}
	return session.Uuid, nil
}

// GetUserSessionList 获取用户会话列表
func (s *sessionService) GetUserSessionList(ownerId string) ([]respond.UserSessionListRespond, error) {
	var sessionList []model.Session
	if res := dao.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("session not found")
			return nil, nil
		} else {
			zlog.Error(res.Error.Error())
			return nil, res.Error
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
	return sessionListRsp, nil
}

// GetGroupSessionList 获取群聊会话列表
func (s *sessionService) GetGroupSessionList(ownerId string) ([]respond.GroupSessionListRespond, error) {
	var sessionList []model.Session
	if res := dao.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("session not found")
			return nil, nil
		} else {
			zlog.Error(res.Error.Error())
			return nil, res.Error
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
	return sessionListRsp, nil
}

// DeleteSession 删除会话
func (s *sessionService) DeleteSession(sessionId string) error {
	if res := dao.GormDB.Where("uuid = ?", sessionId).Delete(&model.Session{}); res.Error != nil {
		zlog.Error(res.Error.Error())
		return res.Error
	}
	return nil
}
