package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/internal/dto/request"
	"github.com/puoxiu/gogochat/internal/service/gorm"
	"github.com/puoxiu/gogochat/pkg/enum/error_info"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

// CreateGroup 创建群聊
func CreateGroup(c *gin.Context) {
	var createGroupReq request.CreateGroupRequest
	if err := c.BindJSON(&createGroupReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": error_info.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.GroupInfoService.CreateGroup(createGroupReq)
	JsonBack(c, message, ret, nil)
}

// LoadMyGroup 获取我创建的群聊
func LoadMyGroup(c *gin.Context) {
	var loadMyGroupReq request.OwnlistRequest
	if err := c.BindJSON(&loadMyGroupReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": error_info.SYSTEM_ERROR,
		})
		return
	}
	message, groupList, ret := gorm.GroupInfoService.LoadMyGroup(loadMyGroupReq.OwnerId)
	JsonBack(c, message, ret, groupList)
}
