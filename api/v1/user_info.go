package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/puoxiu/gogochat/internal/dto/request"
	"github.com/puoxiu/gogochat/internal/service/gorm"
	"github.com/puoxiu/gogochat/pkg/enum/error_info"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

// Register 注册
func Register(c *gin.Context) {
	var registerReq request.RegisterRequest
	if err := c.BindJSON(&registerReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": error_info.SYSTEM_ERROR,
		})
		return
	}
	fmt.Println(registerReq)
	message, userInfoStr, ret := gorm.UserInfoService.Register(c, registerReq)
	JsonBack(c, message, ret, userInfoStr)
}

// Login 登录
func Login(c *gin.Context) {
	var loginReq request.LoginRequest
	if err := c.BindJSON(&loginReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": error_info.SYSTEM_ERROR,
		})
		return
	}
	message, userInfoStr, ret := gorm.UserInfoService.Login(c, loginReq)
	JsonBack(c, message, ret, userInfoStr)
}
