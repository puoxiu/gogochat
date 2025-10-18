package http_server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	v1 "github.com/puoxiu/gogochat/services/chat_service/api/v1"
	"github.com/puoxiu/gogochat/services/chat_service/internal/config"
	// "github.com/puoxiu/gogochat/pkg/ssl"
)
var GE *gin.Engine

func InitHttpServer() {
	GE = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	GE.Use(cors.New(corsConfig))
	// GE.Use(ssl.TlsHandler(config.GetConfig().MainConfig.Host, config.GetConfig().MainConfig.Port))
	GE.Static("/static/avatars", config.AppConfig.StaticSrcConfig.StaticAvatarPath)
	GE.Static("/static/files", config.AppConfig.StaticSrcConfig.StaticFilePath)

	GE.POST("/message/getMessageList", v1.GetMessageList)
	GE.POST("/message/getGroupMessageList", v1.GetGroupMessageList)
	GE.POST("/message/uploadAvatar", v1.UploadAvatar)
	GE.POST("/message/uploadFile", v1.UploadFile)
	GE.POST("/chatroom/getCurContactListInChatRoom", v1.GetCurContactListInChatRoom)
	GE.GET("/wss", v1.WsLogin)
}
