package http_server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/config"
	v1 "github.com/puoxiu/gogochat/services/user_service/api/v1"
	// "github.com/puoxiu/gogochat/pkg/ssl"
)
var GE *gin.Engine

func init() {
	GE = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	GE.Use(cors.New(corsConfig))
	// GE.Use(ssl.TlsHandler(config.GetConfig().MainConfig.Host, config.GetConfig().MainConfig.Port))
	GE.Static("/static/avatars", config.GetConfig().StaticAvatarPath)
	GE.Static("/static/files", config.GetConfig().StaticFilePath)
	GE.POST("/login", v1.Login)
	GE.POST("/register", v1.Register)
	GE.POST("/user/updateUserInfo", v1.UpdateUserInfo)
	GE.POST("/user/getUserInfoList", v1.GetUserInfoList)
	GE.POST("/user/ableUsers", v1.AbleUsers)
	GE.POST("/user/getUserInfo", v1.GetUserInfo)
	GE.POST("/user/disableUsers", v1.DisableUsers)
	GE.POST("/user/deleteUsers", v1.DeleteUsers)
	GE.POST("/user/setAdmin", v1.SetAdmin)
	GE.POST("/user/sendSmsCode", v1.SendSmsCode)
	GE.POST("/user/smsLogin", v1.SmsLogin)
	GE.POST("/contact/getUserList", v1.GetUserList)
	GE.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup)
	GE.POST("/contact/getContactInfo", v1.GetContactInfo)
	GE.POST("/contact/deleteContact", v1.DeleteContact)
	GE.POST("/contact/applyContact", v1.ApplyContact)
	GE.POST("/contact/getNewContactList", v1.GetNewContactList)
	GE.POST("/contact/passContactApply", v1.PassContactApply)
	GE.POST("/contact/blackContact", v1.BlackContact)
	GE.POST("/contact/cancelBlackContact", v1.CancelBlackContact)
	GE.POST("/contact/getAddGroupList", v1.GetAddGroupList)
	GE.POST("/contact/refuseContactApply", v1.RefuseContactApply)
	GE.POST("/contact/blackApply", v1.BlackApply)
}
