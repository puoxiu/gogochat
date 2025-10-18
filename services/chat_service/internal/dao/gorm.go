package dao

import (
	"fmt"

	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/chat_service/internal/model"
	"github.com/puoxiu/gogochat/services/chat_service/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func InitMySQL() {
	conf := config.AppConfig
	user := conf.MySQLConfig.User
	password := conf.MySQLConfig.Password
	host := conf.MySQLConfig.Host
	port := conf.MySQLConfig.Port
	appName := conf.MySQLConfig.DatabaseName
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, appName)
	fmt.Println(dsn)
	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		zlog.Fatal(err.Error())
	}
	err = GormDB.AutoMigrate(
		&model.Message{},
	) 

	if err != nil {
		zlog.Fatal(err.Error())
	}
}
