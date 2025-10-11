package dao

import (
	"fmt"
	"log"

	"github.com/puoxiu/gogochat/config"
	"github.com/puoxiu/gogochat/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func init() {
	conf := config.GetConfig()
	user := conf.User
	password := conf.MysqlConfig.Password
	host := conf.MysqlConfig.Host
	port := conf.MysqlConfig.Port
	appName := conf.AppName
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, appName)
	fmt.Println(dsn)
	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err.Error())
	}
	err = GormDB.AutoMigrate(&model.UserInfo{}, &model.GroupInfo{}) // 自动迁移，如果没有建表，会自动创建对应的表
	if err != nil {
		log.Fatal(err.Error())
	}
}
