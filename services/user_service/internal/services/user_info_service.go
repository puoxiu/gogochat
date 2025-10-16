package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/internal/service/sms"
	"github.com/puoxiu/gogochat/services/user_service/internal/dao"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/respond"
	"github.com/puoxiu/gogochat/services/user_service/internal/model"

	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/enum/user_info/user_status_enum"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"gorm.io/gorm"
)

type userInfoService struct {
}

var UserInfoService = new(userInfoService)

// dao层加不了校验，在service层加
// checkTelephoneValid 检验电话是否有效
func (u *userInfoService) checkTelephoneValid(telephone string) bool {
	pattern := `^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`
	match, err := regexp.MatchString(pattern, telephone)
	if err != nil {
		zlog.Error(err.Error())
	}
	return match
}

// checkEmailValid 校验邮箱是否有效
func (u *userInfoService) checkEmailValid(email string) bool {
	pattern := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	match, err := regexp.MatchString(pattern, email)
	if err != nil {
		zlog.Error(err.Error())
	}
	return match
}

// checkUserIsAdminOrNot 检验用户是否为管理员
func (u *userInfoService) checkUserIsAdminOrNot(user model.UserInfo) int8 {
	return user.IsAdmin
}

// Login 登录
func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
	password := loginReq.Password
	var user model.UserInfo
	res := dao.GormDB.First(&user, "telephone = ?", loginReq.Telephone)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Warn(fmt.Sprintf("用户不存在: telephone=%s", loginReq.Telephone))
			return "用户不存在，请注册", nil, -2
		}
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if user.Password != password {
		zlog.Warn(fmt.Sprintf("密码不正确: telephone=%s", loginReq.Telephone))
		return "密码不正确，请重试", nil, -2
	}

	loginRsp := &respond.LoginRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Birthday:  user.Birthday,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登陆成功", loginRsp, 0
}

// SmsLogin 验证码登录
func (u *userInfoService) SmsLogin(req request.SmsLoginRequest) (string, *respond.LoginRespond, int) {
	var user model.UserInfo
	res := dao.GormDB.First(&user, "telephone = ?", req.Telephone)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Warn(fmt.Sprintf("用户不存在: telephone=%s", req.Telephone))
			return "用户不存在，请注册", nil, -2
		}
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	} 

	key := "auth_code_" + req.Telephone
	code, err := cache.GetGlobalCache().GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if code != req.SmsCode {
		zlog.Warn(fmt.Sprintf("验证码不正确: telephone=%s", req.Telephone))
		return "验证码不正确，请重试", nil, -2
	} else {
		if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	loginRsp := &respond.LoginRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Birthday:  user.Birthday,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登陆成功", loginRsp, 0
}

// SendSmsCode 发送短信验证码 - 验证码登录
func (u *userInfoService) SendSmsCode(telephone string) (string, int) {
	return sms.VerificationCode(telephone)
}

// checkTelephoneExist 检查手机号是否存在
func (u *userInfoService) checkTelephoneExist(telephone string) (string, int) {
	var user model.UserInfo
	// gorm默认排除软删除，所以翻译过来的select语句是SELECT * FROM `user_info` WHERE telephone = '18089596095' AND `user_info`.`deleted_at` IS NULL ORDER BY `user_info`.`id` LIMIT 1
	if res := dao.GormDB.Where("telephone = ?", telephone).First(&user); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Warn(fmt.Sprintf("手机号不存在: telephone=%s", telephone))
			return "", -2
		}
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	zlog.Info(fmt.Sprintf("手机号查找成功: telephone=%s", telephone))
	return "该电话已经存在", 0
}

// Register 注册，返回(message, register_respond_string, error)
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.RegisterRespond, int) {
	key := "auth_code_" + registerReq.Telephone
	code, err := cache.GetGlobalCache().GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if code != registerReq.SmsCode {
		zlog.Warn(fmt.Sprintf("验证码不正确: telephone=%s", registerReq.Telephone))
		return "验证码不正确，请重试", nil, -2
	} else {
		if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	// 判断电话是否已经被注册过了
	message, ret := u.checkTelephoneExist(registerReq.Telephone)
	if ret != 0 {
		return message, nil, ret
	}
	var newUser model.UserInfo
	newUser.Uuid = "U" + random.GetNowAndLenRandomString(11)
	newUser.Telephone = registerReq.Telephone
	newUser.Password = registerReq.Password
	newUser.Nickname = registerReq.Nickname
	newUser.Avatar = "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png"
	newUser.CreatedAt = time.Now()
	newUser.IsAdmin = u.checkUserIsAdminOrNot(newUser)
	newUser.Status = user_status_enum.NORMAL
	// 手机号验证，最后一步才调用api，省钱hhh
	//err := sms.VerificationCode(registerReq.Telephone)
	//if err != nil {
	//	zlog.Error(err.Error())
	//	return "", err
	//}

	res := dao.GormDB.Create(&newUser)
	if res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	// 注册成功，chat client建立
	//if err := chat.NewClientInit(c, newUser.Uuid); err != nil {
	//	return "", err
	//}
	registerRsp := &respond.RegisterRespond{
		Uuid:      newUser.Uuid,
		Telephone: newUser.Telephone,
		Nickname:  newUser.Nickname,
		Email:     newUser.Email,
		Avatar:    newUser.Avatar,
		Gender:    newUser.Gender,
		Birthday:  newUser.Birthday,
		Signature: newUser.Signature,
		IsAdmin:   newUser.IsAdmin,
		Status:    newUser.Status,
	}
	year, month, day := newUser.CreatedAt.Date()
	registerRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "注册成功", registerRsp, 0
}

// UpdateUserInfo 修改用户信息
// 某用户修改了信息，可能会影响contact_user_list，不需要删除redis的contact_user_list，timeout之后会自己更新
// 但是需要更新redis的user_info，因为可能影响用户搜索
func (u *userInfoService) UpdateUserInfo(updateReq request.UpdateUserInfoRequest) (string, int) {
	var user model.UserInfo
	if res := dao.GormDB.First(&user, "uuid = ?", updateReq.Uuid); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if updateReq.Email != "" {
		user.Email = updateReq.Email
	}
	if updateReq.Nickname != "" {
		user.Nickname = updateReq.Nickname
	}
	if updateReq.Birthday != "" {
		user.Birthday = updateReq.Birthday
	}
	if updateReq.Signature != "" {
		user.Signature = updateReq.Signature
	}
	if updateReq.Avatar != "" {
		user.Avatar = updateReq.Avatar
	}
	if res := dao.GormDB.Save(&user); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if err := cache.GetGlobalCache().DelKeysWithPattern("user_info_" + updateReq.Uuid); err != nil {
		zlog.Error(err.Error())
	}
	return "修改用户信息成功", 0
}

// GetUserInfoList 获取用户列表除了ownerId之外 - 管理员
// 管理员少，而且如果用户更改了，那么管理员会一直频繁删除redis，更新redis，比较麻烦，所以管理员暂时不使用redis缓存
func (u *userInfoService) GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int) {
	// redis中没有数据，从数据库中获取
	var users []model.UserInfo
	// 获取所有的用户
	if res := dao.GormDB.Unscoped().Where("uuid != ?", ownerId).Find(&users); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	var rsp []respond.GetUserListRespond
	for _, user := range users {
		rp := respond.GetUserListRespond{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Status:    user.Status,
			IsAdmin:   user.IsAdmin,
		}
		if user.DeletedAt.Valid {
			rp.IsDeleted = true
		} else {
			rp.IsDeleted = false
		}
		rsp = append(rsp, rp)
	}
	return "获取用户列表成功", rsp, 0
}

// AbleUsers 启用用户--解封
func (u *userInfoService) AbleUsers(uuidList []string) (string, int) {
    res := dao.GormDB.Model(model.UserInfo{}).
        Where("uuid in (?)", uuidList).
        Update("status", user_status_enum.NORMAL)
    if res.Error != nil {
        zlog.Error("批量启用用户失败: " + res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if res.RowsAffected == 0 {
        zlog.Info("未找到可启用的用户")
        return "未找到可启用的用户", -2
    }
    zlog.Info(fmt.Sprintf("成功启用 %d 个用户", res.RowsAffected))

	// todo cache
	for _, uuid := range uuidList {
		if err := cache.GetGlobalCache().DelKeyIfExists("contact_user_list_" + uuid); err != nil {
			zlog.Warn(fmt.Sprintf("清理用户自身联系人缓存失败: uuid=%s, err=%v", uuid, err))
		}
		if err := cache.GetGlobalCache().DelKeyIfExists("user_info_" + uuid); err != nil {
			zlog.Warn(fmt.Sprintf("清理用户自身信息缓存失败: uuid=%s, err=%v", uuid, err))
		}
	}

	return fmt.Sprintf("成功启用 %d 个用户", res.RowsAffected), 0
}

// DisableUsers 禁用用户--封号
func (u *userInfoService) DisableUsers(uuidList []string) (string, int) {
    res := dao.GormDB.Model(model.UserInfo{}).
        Where("uuid in (?)", uuidList).
        Update("status", user_status_enum.DISABLE)
    
	if res.Error != nil {
        zlog.Error("批量禁用用户失败: " + res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if res.RowsAffected == 0 {
        zlog.Info("未找到可禁用的用户")
        return "未找到可禁用的用户", -2
    }
	zlog.Info(fmt.Sprintf("成功禁用 %d 个用户", res.RowsAffected))

	// todo cache
	for _, uuid := range uuidList {
		if err := cache.GetGlobalCache().DelKeyIfExists("contact_user_list_" + uuid); err != nil {
			zlog.Warn(fmt.Sprintf("清理用户自身联系人缓存失败: uuid=%s, err=%v", uuid, err))
		}
		if err := cache.GetGlobalCache().DelKeyIfExists("user_info_" + uuid); err != nil {
			zlog.Warn(fmt.Sprintf("清理用户自身信息缓存失败: uuid=%s, err=%v", uuid, err))
		}
	}

	return fmt.Sprintf("成功禁用 %d 个用户", res.RowsAffected), 0
}

// DeleteUsers 删除用户
func (u *userInfoService) DeleteUser(uuid string) (string, int) {
	//软删除用户
	res := dao.GormDB.Delete(&model.UserInfo{}, "uuid = ?", uuid)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info(res.Error.Error())
			return "未找到可删除的用户", -2
		} else {
			zlog.Error(fmt.Sprintf("删除用户数据库失败: uuid=%s, err=%v", uuid, res.Error))
			return constants.SYSTEM_ERROR, -1
		}
	}
	if res.RowsAffected == 0 {
		zlog.Info("未找到可删除的用户")
		return "未找到可删除的用户", -2
	}
	zlog.Info(fmt.Sprintf("用户软删除成功:uuid=%s", uuid))


	// 删除联系人
	var contactList []model.UserContact
	if res := dao.GormDB.Where("user_id = ?", uuid).Find(&contactList); res.Error != nil {
		zlog.Error(fmt.Sprintf("联系人查询失败: uuid=%s, err=%v", uuid, res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	// 批量软删除联系人（避免循环单条删除，提升效率）
	if len(contactList) > 0 {
		IDs := make([]int64, 0, len(contactList))
		for _, contact := range contactList {
			IDs = append(IDs, contact.Id)
		}
		contactDelRes := dao.GormDB.Delete(&model.UserContact{}, "id in (?)", IDs)
		if contactDelRes.Error != nil {
			zlog.Error(fmt.Sprintf("联系人批量软删除失败:uuid=%s, err=%v", uuid, contactDelRes.Error))
			return constants.SYSTEM_ERROR, -1
		}
		zlog.Info(fmt.Sprintf("成功删除 %d 个联系人", contactDelRes.RowsAffected))
	} else {
		zlog.Info(fmt.Sprintf("无关联联系人可删除:uuid=%s", uuid))
	}

	// 软删除会话--调用rpc
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败: err=%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	for _, contact := range contactList {
		resp1 := sessionClient.DeleteSessionsByUsers(contact.UserId, contact.ContactId)
		if resp1.Code != 0 {
			zlog.Warn(fmt.Sprintf("删除会话失败: user_id=%s, contact_id=%s, err=%s", contact.UserId, contact.ContactId, resp1.Message))
		}
		resp2 := sessionClient.DeleteSessionsByUsers(contact.ContactId, contact.UserId)
		if resp2.Code != 0 {
			zlog.Warn(fmt.Sprintf("删除会话失败: user_id=%s, contact_id=%s, err=%s", contact.ContactId, contact.UserId, resp2.Message))
		}
	}
	
	// todo cache
	if err := cache.GetGlobalCache().DelKeyIfExists("contact_user_list_" + uuid); err != nil {
		zlog.Warn(fmt.Sprintf("清理用户自身联系人缓存失败: uuid=%s, err=%v", uuid, err))
	}
	if err := cache.GetGlobalCache().DelKeyIfExists("user_info_" + uuid); err != nil {
		zlog.Warn(fmt.Sprintf("清理用户自身信息缓存失败: uuid=%s, err=%v", uuid, err))
	}

	return "删除用户成功", 0
}

// GetUserInfo 获取用户信息
func (u *userInfoService) GetUserInfo(uuid string) (string, *respond.GetUserInfoRespond, int) {
	// redis
	zlog.Info(uuid)
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("user_info_" + uuid)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info(fmt.Sprintf("用户缓存未命中查询数据库 uuid=%s", uuid))
			var user model.UserInfo
			res := dao.GormDB.Where("uuid = ?", uuid).First(&user)
			if res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Info(res.Error.Error())
					return "用户不存在", nil, -2
				} else {
					zlog.Error(fmt.Sprintf("查询用户数据库失败:uuid=%s, err=%v", uuid, res.Error))
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			
			rsp := respond.GetUserInfoRespond{
				Uuid:      user.Uuid,
				Telephone: user.Telephone,
				Nickname:  user.Nickname,
				Avatar:    user.Avatar,
				Birthday:  user.Birthday,
				Email:     user.Email,
				Gender:    user.Gender,
				Signature: user.Signature,
				CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
				IsAdmin:   user.IsAdmin,
				Status:    user.Status,
			}
			rspString, err := json.Marshal(rsp)
			if err != nil {
				zlog.Error(err.Error())
			}
			if err := cache.GetGlobalCache().SetKeyEx("user_info_"+uuid, string(rspString), constants.REDIS_TIMEOUT*time.Hour); err != nil {
				zlog.Warn(fmt.Sprintf("用户缓存写入失败: uuid=%s, err=%v", uuid, err))
			} else {
				zlog.Info(fmt.Sprintf("用户缓存写入成功: uuid=%s", uuid))
			}
			return "获取用户信息成功", &rsp, 0
		} else {
			zlog.Error(fmt.Sprintf("查询用户缓存失败: uuid=%s, err=%v", uuid, err))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp respond.GetUserInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(fmt.Sprintf("解析用户缓存失败: uuid=%s, err=%v", uuid, err))
		return constants.SYSTEM_ERROR, nil, -1
	}
	zlog.Info(fmt.Sprintf("用户缓存命中: uuid=%s", uuid))
	return "获取用户信息成功", &rsp, 0
}

// SetAdmin 设置管理员
func (u *userInfoService) SetAdmin(uuidList []string, isAdmin int8) (string, int) {
	var users []model.UserInfo
	if res := dao.GormDB.Where("uuid = (?)", uuidList).Find(&users); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, user := range users {
		user.IsAdmin = isAdmin
		if res := dao.GormDB.Save(&user); res.Error != nil {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}
	return "设置管理员成功", 0
}
