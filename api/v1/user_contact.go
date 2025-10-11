package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/internal/dto/request"
	"github.com/puoxiu/gogochat/internal/service/gorm"
	"log"
)

// GetUserList 获取联系人列表
func GetUserList(c *gin.Context) {
	var myUserListReq request.OwnlistRequest
	if err := c.BindJSON(&myUserListReq); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
	}
	message, userList, err := gorm.UserContactService.GetUserList(myUserListReq.OwnerId)
	if message == "" && err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
	} else if message != "" && err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": message,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "get userlist success",
			"data":    userList,
		})
	}
}

// LoadMyJoinedGroup 获取我加入的群聊
func LoadMyJoinedGroup(c *gin.Context) {
	var loadMyJoinedGroupReq request.OwnlistRequest
	if err := c.BindJSON(&loadMyJoinedGroupReq); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
		return
	}
	groupList, err := gorm.UserContactService.LoadMyJoinedGroup(loadMyJoinedGroupReq.OwnerId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "load my joined group success",
		"data":    groupList,
	})
}

// GetContactInfo 获取联系人信息
func GetContactInfo(c *gin.Context) {
	var getContactInfoReq request.GetContactInfoRequest
	if err := c.BindJSON(&getContactInfoReq); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
		return
	}
	log.Println(getContactInfoReq)
	message, contactInfo, err := gorm.UserContactService.GetContactInfo(getContactInfoReq.ContactId)
	if message == "" && err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
	} else if message != "" && err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": message,
		})
	} else {
		log.Println(contactInfo)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "get contact name success",
			"data":    contactInfo,
		})
	}
}

// DeleteContact 删除联系人
func DeleteContact(c *gin.Context) {
	var deleteContactReq request.DeleteContactRequest
	if err := c.BindJSON(&deleteContactReq); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
		return
	}
	err := gorm.UserContactService.DeleteContact(deleteContactReq.OwnerId, deleteContactReq.ContactId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  400,
			"error": err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "delete contact success",
		})
	}
}
