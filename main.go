package main

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/puoxiu/gogochat/api/v1"
	"github.com/puoxiu/gogochat/pkg/zlog"
)



func main() {
	r := gin.Default()
	r.POST("/login", v1.Login)
	r.POST("/register", v1.Register)
	if err := r.Run("127.0.0.1:8080"); err != nil {
		zlog.Fatal("server running fault")
		return
	}
}
