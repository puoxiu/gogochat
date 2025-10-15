package http_server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	v1 "github.com/puoxiu/gogochat/services/session_service/api/v1"
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
	GE.POST("/session/openSession", v1.OpenSession)
	GE.POST("/session/getUserSessionList", v1.GetUserSessionList)
	GE.POST("/session/getGroupSessionList", v1.GetGroupSessionList)
	GE.POST("/session/deleteSession", v1.DeleteSession)
	GE.POST("/session/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed)
}
