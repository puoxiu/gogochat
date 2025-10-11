package main

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	v1 "github.com/puoxiu/gogochat/api/v1"
	"github.com/puoxiu/gogochat/config"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

func main() {
	// 1. 初始化 Gin 引擎（默认包含日志和恢复中间件）
	r := gin.Default()

	// 2. 配置跨域中间件（允许前端与后端跨域通信）
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有源（开发环境）
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // 允许的 HTTP 方法
		AllowHeaders:     []string{"Content-Type", "Authorization"}, // 允许的请求头
		ExposeHeaders:    []string{"X-Custom-Header"}, // 允许前端获取的响应头
		AllowCredentials: true, // 允许携带 Cookie 等认证信息
	}))

	// 3. 注册 API 路由（v1 版本接口）
	registerRoutes(r)

	// 4. 从配置文件获取服务地址和端口
	conf := config.GetConfig()
	serverAddr := fmt.Sprintf("%s:%d", conf.MainConfig.Host, conf.MainConfig.Port)

	// 5. 启动 HTTP 服务
	zlog.Info(fmt.Sprintf("server starting on %s", serverAddr))
	if err := r.Run(serverAddr); err != nil {
		zlog.Fatal(fmt.Sprintf("server failed to start: %v", err))
	}
}

// registerRoutes 注册所有 API 路由，分离路由配置与主逻辑，提高可读性
func registerRoutes(r *gin.Engine) {
	// 用户相关接口
	r.POST("/login", v1.Login)         // 用户登录
	r.POST("/register", v1.Register)   // 用户注册

	// 联系人相关接口
	r.POST("/contact/getUserList", v1.GetUserList)               // 获取联系人列表
	r.POST("/contact/getContactInfo", v1.GetContactInfo)         // 获取联系人详情
	r.POST("/contact/deleteContact", v1.DeleteContact)           // 删除联系人
	r.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup)   // 获取已加入的群组列表

	// 群组相关接口
	r.POST("/group/createGroup", v1.CreateGroup)   // 创建群组
	r.POST("/group/loadMyGroup", v1.LoadMyGroup)   // 获取我创建的群组列表

	// 会话相关接口
	r.POST("/session/openSession", v1.OpenSession)               // 打开会话（单聊/群聊）
	r.POST("/session/getUserSessionList", v1.GetUserSessionList) // 获取单聊会话列表
	r.POST("/session/getGroupSessionList", v1.GetGroupSessionList) // 获取群聊会话列表
	r.POST("/session/deleteSession", v1.DeleteSession)           // 删除会话
}