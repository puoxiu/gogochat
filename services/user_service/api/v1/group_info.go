package v1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/user_service/internal/services"
)

// CreateGroup 创建群聊
func CreateGroup(c *gin.Context) {
	var createGroupReq request.CreateGroupRequest
	if err := c.BindJSON(&createGroupReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.CreateGroup(createGroupReq)
	JsonBack(c, message, ret, nil)
}

// LoadMyGroup 获取我创建的群聊
func LoadMyGroup(c *gin.Context) {
	var loadMyGroupReq request.OwnlistRequest
	if err := c.BindJSON(&loadMyGroupReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, groupList, ret := services.GroupInfoService.LoadMyGroup(loadMyGroupReq.OwnerId)
	JsonBack(c, message, ret, groupList)
}

// CheckGroupAddMode 检查群聊加群方式
func CheckGroupAddMode(c *gin.Context) {
	var req request.CheckGroupAddModeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, addMode, ret := services.GroupInfoService.CheckGroupAddMode(req.GroupId)
	JsonBack(c, message, ret, addMode)
}

// EnterGroupDirectly 直接进群
func EnterGroupDirectly(c *gin.Context) {
	var req request.EnterGroupDirectlyRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.EnterGroupDirectly(req.OwnerId, req.ContactId)
	JsonBack(c, message, ret, nil)
}

// LeaveGroup 退群
func LeaveGroup(c *gin.Context) {
	var req request.LeaveGroupRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.LeaveGroup(req.UserId, req.GroupId)
	JsonBack(c, message, ret, nil)
}

// DismissGroup 解散群聊
func DismissGroup(c *gin.Context) {
	var req request.DismissGroupRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.DismissGroup(req.OwnerId, req.GroupId)
	JsonBack(c, message, ret, nil)
}

// GetGroupInfo 获取群聊详情
func GetGroupInfo(c *gin.Context) {
	var req request.GetGroupInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, groupInfo, ret := services.GroupInfoService.GetGroupInfo(req.GroupId)
	JsonBack(c, message, ret, groupInfo)
}


// UpdateGroupInfo 更新群聊消息
func UpdateGroupInfo(c *gin.Context) {
	var req request.UpdateGroupInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.UpdateGroupInfo(req)
	JsonBack(c, message, ret, nil)
}

// GetGroupMemberList 获取群聊成员列表
func GetGroupMemberList(c *gin.Context) {
	var req request.GetGroupMemberListRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, groupMemberList, ret := services.GroupInfoService.GetGroupMemberList(req.GroupId)
	JsonBack(c, message, ret, groupMemberList)
}

// RemoveGroupMembers 移除群聊成员
func RemoveGroupMembers(c *gin.Context) {
	log.Println("===========移除群聊成员请求:")
	var req request.RemoveGroupMembersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := services.GroupInfoService.RemoveGroupMembers(req)
	JsonBack(c, message, ret, nil)
}
