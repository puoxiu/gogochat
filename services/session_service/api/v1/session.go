package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/session_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/session_service/internal/services"
)

// OpenSession 打开会话
func OpenSession(c *gin.Context) {
	var openSessionReq request.OpenSessionRequest
	if err := c.BindJSON(&openSessionReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, sessionId, code := services.SessionService.OpenSession(openSessionReq.SendId, openSessionReq.ReceiveId)
	JsonBack(c, message, code, sessionId)
}

// GetUserSessionList 获取用户会话列表
func GetUserSessionList(c *gin.Context) {
	var getUserSessionListReq request.OwnlistRequest
	if err := c.BindJSON(&getUserSessionListReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, sessionList, code := services.SessionService.GetUserSessionList(getUserSessionListReq.OwnerId)
	JsonBack(c, message, code, sessionList)
}

// GetGroupSessionList 获取群聊会话列表
func GetGroupSessionList(c *gin.Context) {
	var getGroupListReq request.OwnlistRequest
	if err := c.BindJSON(&getGroupListReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, groupList, code := services.SessionService.GetGroupSessionList(getGroupListReq.OwnerId)
	JsonBack(c, message, code, groupList)
}

// DeleteSession 删除会话
func DeleteSession(c *gin.Context) {
	var deleteSessionReq request.DeleteSessionRequest
	if err := c.BindJSON(&deleteSessionReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, code := services.SessionService.DeleteSession(deleteSessionReq.OwnerId, deleteSessionReq.SessionId)
	JsonBack(c, message, code, nil)
}

// CheckOpenSessionAllowed 检查是否可以打开会话
func CheckOpenSessionAllowed(c *gin.Context) {
	var req request.CreateSessionRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, res, code := services.SessionService.CheckOpenSessionAllowed(req.SendId, req.ReceiveId)
	JsonBack(c, message, code, res)
}
