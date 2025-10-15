package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func JsonBack(c *gin.Context, message string, code int, data interface{}) {
	if code == 1 {
		// 成功
		if data != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
				"data":    data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
			})
		}
	} 
	
	if code == 0 {
		// 请求数据出错/不存在
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": message,
		})
	}

	if code == -1 {
		// 服务端错误
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": message,
		})
		return
	}
}
