package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/enum/contact/contact_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/contact/contact_type_enum"
	"github.com/puoxiu/gogochat/pkg/enum/contact_apply/contact_apply_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/group_info/group_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/user_info/user_status_enum"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/user_service/internal/dao"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/respond"
	"github.com/puoxiu/gogochat/services/user_service/internal/model"
	"gorm.io/gorm"
)

type userContactService struct {
}

var UserContactService = new(userContactService)

// GetUserList 获取用户联系人列表（符合逻辑：保留禁用/单向删除联系人，仅互动时限制）-✅
func (u *userContactService) GetUserList(uuid string) (string, []respond.MyUserListRespond, int) {
	cacheKey := "contact_user_list_" + uuid
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr(cacheKey)
	if err != nil {
		// 缓存未命中，走数据库查询
		if errors.Is(err, redis.Nil) {
			var contactList []model.UserContact
            contactRes := dao.GormDB.
                Order("created_at DESC"). // 按添加时间倒序，最新的在前面
                Where("user_id = ? AND status != ?", uuid, 3). // 仅排除当前用户自己删的联系人
                Find(&contactList)
			if contactRes.Error != nil {
                if errors.Is(contactRes.Error, gorm.ErrRecordNotFound) {
                    zlog.Info(fmt.Sprintf("用户 %s 暂无联系人", uuid))
                    return "目前不存在联系人", nil, -2
                }
                zlog.Error(fmt.Sprintf("查询用户 %s 联系人失败: %v", uuid, contactRes.Error))
                return constants.SYSTEM_ERROR, nil, -1
			}
			var userListRsp []respond.MyUserListRespond
			for _, contact := range contactList {
				if contact.ContactType != contact_type_enum.User {
					continue
				}
				// 
				var contactUser model.UserInfo
                userRes := dao.GormDB.
                    Where("uuid = ?", contact.ContactId).
                    First(&contactUser)
				if userRes.Error != nil {
					zlog.Warn(fmt.Sprintf("查询用户 %s 联系人 %s 失败: %v", uuid, contact.ContactId, userRes.Error))
					continue
				}
				userListRsp = append(userListRsp, respond.MyUserListRespond{
					UserId:   contactUser.Uuid,
					UserName: contactUser.Nickname,
					Avatar:   contactUser.Avatar,
				})
			}
			// 缓存用户列表
			if len(userListRsp) > 0 {
				rspString, err := json.Marshal(userListRsp)
				if err != nil {
					zlog.Error(err.Error())
				} else {
					if cacheErr := cache.GetGlobalCache().SetKeyEx(cacheKey, string(rspString), time.Hour*constants.REDIS_TIMEOUT); cacheErr != nil {
						zlog.Warn(fmt.Sprintf("联系人缓存写入失败: %v", cacheErr))
					} else {
						zlog.Info(fmt.Sprintf("联系人缓存写入成功: %s", cacheKey))
					}
				}
			}
			return "获取用户列表成功", userListRsp, 0
		}
		zlog.Error(fmt.Sprintf("查询用户 %s 联系人缓存失败: %v", uuid, err))
		return constants.SYSTEM_ERROR, nil, -1
	}
	// 从缓存中获取用户列表
	var rsp []respond.MyUserListRespond
	unmarshalErr := json.Unmarshal([]byte(rspString), &rsp)	
    if unmarshalErr != nil {
        zlog.Error(fmt.Sprintf("联系人缓存反序列化失败: %v", unmarshalErr))
        // 缓存损坏时，重新走数据库逻辑（降级策略）
        return u.GetUserList(uuid)
    }
	zlog.Info(fmt.Sprintf("用户 %s 联系人缓存命中，共 %d 条", uuid, len(rsp)))
	return "获取用户列表成功", rsp, 0
}

// LoadMyJoinedGroup 获取我加入的群聊列表 -✅
func (u *userContactService) LoadMyJoinedGroup(ownerId string) (string, []respond.LoadMyJoinedGroupRespond, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("my_joined_group_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var contactList []model.UserContact
			// 没有退群，也没有被踢出群聊
			if res := dao.GormDB.Order("created_at DESC").Where("user_id = ? AND status != 6 AND status != 7", ownerId).Find(&contactList); res.Error != nil {
				// 不存在不是业务问题，用Info，return 0
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Info("目前不存在加入的群聊")
					return "目前不存在加入的群聊", nil, 0
				} else {
					zlog.Error(fmt.Sprintf("查询用户 %s 加入的群聊失败: %v", ownerId, res.Error))
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var groupList []model.GroupInfo
			for _, contact := range contactList {
				if contact.ContactId[0] == 'G' {
					// 获取群聊信息
					var group model.GroupInfo
					if res := dao.GormDB.First(&group, "uuid = ?", contact.ContactId); res.Error != nil {
						zlog.Error(fmt.Sprintf("查询群聊 %s 失败: %v", contact.ContactId, res.Error))
						return constants.SYSTEM_ERROR, nil, -1
					}
					// 群没被删除，同时群主不是自己
					// 群主删除或admin删除群聊，status为7，即被踢出群聊，所以不用判断群是否被删除，删除了到不了这步
					if group.OwnerId != ownerId {
						groupList = append(groupList, group)
					}
				}
			}
			var groupListRsp []respond.LoadMyJoinedGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, respond.LoadMyJoinedGroupRespond{
					GroupId:   group.Uuid,
					GroupName: group.Name,
					Avatar:    group.Avatar,
				})
			}
			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zlog.Error(fmt.Sprintf("加入群聊缓存序列化失败: %v", err))
				return constants.SYSTEM_ERROR, nil, -1
			}
			if err := cache.GetGlobalCache().SetKeyEx("my_joined_group_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(fmt.Sprintf("加入群聊缓存写入失败: %v", err))
			}
			return "获取加入群成功", groupListRsp, 0
		} else {
			zlog.Error(fmt.Sprintf("查询用户 %s 加入的群聊缓存失败: %v", ownerId, err))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.LoadMyJoinedGroupRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}
	return "获取加入群成功", rsp, 0
}

// GetContactInfo 获取联系人信息 -✅
// 调用这个接口的前提是该联系人没有处在删除或被删除，或者该用户还在群聊中
// redis todo
func (u *userContactService) GetContactInfo(contactId string) (string, respond.GetContactInfoRespond, int) {
	if contactId[0] == 'G' {
		var group model.GroupInfo
		if res := dao.GormDB.First(&group, "uuid = ?", contactId); res.Error != nil {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}
		// 没被禁用
		if group.Status != group_status_enum.DISABLE {
			return "获取联系人信息成功", respond.GetContactInfoRespond{
				ContactId:        group.Uuid,
				ContactName:      group.Name,
				ContactAvatar:    group.Avatar,
				ContactNotice:    group.Notice,
				ContactAddMode:   group.AddMode,
				ContactMembers:   group.Members,
				ContactMemberCnt: group.MemberCnt,
				ContactOwnerId:   group.OwnerId,
			}, 0
		} else {
			zlog.Error("该群聊处于禁用状态")
			return "该群聊处于禁用状态", respond.GetContactInfoRespond{}, -2
		}
	} else {
		var user model.UserInfo
		if res := dao.GormDB.First(&user, "uuid = ?", contactId); res.Error != nil {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}
		log.Println(user)
		if user.Status != user_status_enum.DISABLE {
			return "获取联系人信息成功", respond.GetContactInfoRespond{
				ContactId:        user.Uuid,
				ContactName:      user.Nickname,
				ContactAvatar:    user.Avatar,
				ContactBirthday:  user.Birthday,
				ContactEmail:     user.Email,
				ContactPhone:     user.Telephone,
				ContactGender:    user.Gender,
				ContactSignature: user.Signature,
			}, 0
		} else {
			zlog.Info("该用户处于禁用状态")
			return "该用户处于禁用状态", respond.GetContactInfoRespond{}, -2
		}
	}
}

// DeleteContact 删除联系人（只包含用户）--单向删除机制 -✅
// 此部分需要注意服务拆分：session服务接口的调用，删除会话
func (u *userContactService) DeleteContact(ownerId, contactId string) (string, int) {
	// status改变为删除
	// 1. 更新用户联系人关系（用户服务职责）
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", ownerId, contactId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     contact_status_enum.DELETE,
	}); res.Error != nil {
		zlog.Error(fmt.Sprintf("删除用户联系人关系失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}

	// 2. 调用会话服务删除关联会话（解耦核心）todo
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败：%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	resp := sessionClient.DeleteSessionsByUsers(ownerId, contactId)
	if resp.Code != 0 {
		zlog.Warn(fmt.Sprintf("删除会话失败：%v", resp.Message))
		return resp.Message, int(resp.Code)
	}
		
	// 更新联系人申请记录（用户服务职责，保留）
	// 联系人添加的记录得删，这样之后再添加就看新的申请记录，如果申请记录结果是拉黑就没法再添加，如果是拒绝可以再添加
	if res := dao.GormDB.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if err := cache.GetGlobalCache().DelKeysWithPattern("contact_user_list_" + ownerId); err != nil {
		zlog.Error(fmt.Sprintf("删除用户联系人缓存失败：%v", err))
	}
	return "删除联系人成功", 0
}

// ApplyContact 申请添加联系人(包括群聊) -✅
func (u *userContactService) ApplyContact(req request.ApplyContactRequest) (string, int) {
	if req.ContactId[0] == 'U' {
		var contactApply model.ContactApply
		if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", req.OwnerId, req.ContactId).First(&contactApply); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				contactApply = model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.User,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}
				if res := dao.GormDB.Create(&contactApply); res.Error != nil {
					// 不存在记录，则新建就好了
					zlog.Error(fmt.Sprintf("创建联系人申请记录失败：%v", res.Error))
					return constants.SYSTEM_ERROR, -1
				}
			} else {
				zlog.Error(fmt.Sprintf("查询联系人申请记录失败：%v", res.Error))
				return constants.SYSTEM_ERROR, -1
			}
		}
		// 如果存在申请记录，先看看有没有被拉黑
		if contactApply.Status == contact_apply_status_enum.BLACK {
			return "对方已将你拉黑", -2
		}
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contact_apply_status_enum.PENDING

		if res := dao.GormDB.Save(&contactApply); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	} else if req.ContactId[0] == 'G' {
		var group model.GroupInfo
		if res := dao.GormDB.First(&group, "uuid = ?", req.ContactId); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("群聊不存在")
				return "群聊不存在", -2
			} else {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		if group.Status == group_status_enum.DISABLE {
			zlog.Info("群聊已被禁用")
			return "群聊已被禁用", -2
		}
		var contactApply model.ContactApply
		if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", req.OwnerId, req.ContactId).First(&contactApply); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				contactApply = model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.Group,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}
				if res := dao.GormDB.Create(&contactApply); res.Error != nil {
					zlog.Error(fmt.Sprintf("创建群聊联系申请记录失败：%v", res.Error))
					return constants.SYSTEM_ERROR, -1
				}
			} else {
				zlog.Error(fmt.Sprintf("查询群聊申请记录失败：%v", res.Error))
				return constants.SYSTEM_ERROR, -1
			}
		}
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contact_apply_status_enum.PENDING

		if res := dao.GormDB.Save(&contactApply); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存群聊申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	} else {
		return "用户/群聊不存在", -2
	}
}

// GetNewContactList 获取新收到的联系人申请列表 -✅
func (u *userContactService) GetNewContactList(ownerId string) (string, []respond.NewContactListRespond, int) {
	var contactApplyList []model.ContactApply
	if res := dao.GormDB.Where("contact_id = ? AND status = ?", ownerId, contact_apply_status_enum.PENDING).Find(&contactApplyList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("没有收到新的联系人申请")
			return "没有收到新的联系人申请", nil, 0
		} else {
			zlog.Error(fmt.Sprintf("查询联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.NewContactListRespond
	// 所有contact都没被删除
	for _, contactApply := range contactApplyList {
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}
		newContact := respond.NewContactListRespond{
			ContactId: contactApply.Uuid,
			Message:   message,
		}
		var user model.UserInfo
		if res := dao.GormDB.First(&user, "uuid = ?", contactApply.UserId); res.Error != nil {
			return constants.SYSTEM_ERROR, nil, -1
		}
		newContact.ContactId = user.Uuid
		newContact.ContactName = user.Nickname
		newContact.ContactAvatar = user.Avatar
		rsp = append(rsp, newContact)
	}
	return "获取成功", rsp, 0
}

// GetAddGroupList 获取新的加群列表 -✅
// 前端已经判断调用接口的用户是群主，也只有群主才能调用这个接口
func (u *userContactService) GetAddGroupList(groupId string) (string, []respond.AddGroupListRespond, int) {
	var contactApplyList []model.ContactApply
	if res := dao.GormDB.Where("contact_id = ? AND status = ?", groupId, contact_apply_status_enum.PENDING).Find(&contactApplyList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("没有在申请的联系人")
			return "没有在申请的联系人", nil, 0
		} else {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.AddGroupListRespond
	for _, contactApply := range contactApplyList {
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}
		newContact := respond.AddGroupListRespond{
			ContactId: contactApply.Uuid,
			Message:   message,
		}
		var user model.UserInfo
		if res := dao.GormDB.First(&user, "uuid = ?", contactApply.UserId); res.Error != nil {
			return constants.SYSTEM_ERROR, nil, -1
		}
		newContact.ContactId = user.Uuid
		newContact.ContactName = user.Nickname
		newContact.ContactAvatar = user.Avatar
		rsp = append(rsp, newContact)
	}
	return "获取成功", rsp, 0
}

// PassContactApply 通过联系人申请 -✅
func (u *userContactService) PassContactApply(ownerId string, contactId string) (string, int) {
	// ownerId 如果是用户的话就是登录用户，如果是群聊的话就是群聊id
	var contactApply model.ContactApply
	if res := dao.GormDB.Where("contact_id = ? AND user_id = ?", ownerId, contactId).First(&contactApply); res.Error != nil {
		zlog.Error(fmt.Sprintf("查询联系人申请记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	if ownerId[0] == 'U' {
		var user model.UserInfo
		if res := dao.GormDB.Where("uuid = ?", contactId).Find(&user); res.Error != nil {
			zlog.Error(fmt.Sprintf("查询用户信息失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		if user.Status == user_status_enum.DISABLE {
			zlog.Warn(fmt.Sprintf("用户 %s 已被禁用, 无法添加为联系人", user.Uuid))
			return "该用户已被禁用", -2
		}
		contactApply.Status = contact_apply_status_enum.AGREE
		if res := dao.GormDB.Save(&contactApply); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		newContact := model.UserContact{
			UserId:      ownerId,
			ContactId:   contactId,
			ContactType: contact_type_enum.User,     // 用户
			Status:      contact_status_enum.NORMAL, // 正常
			CreatedAt:   time.Now(),
			UpdateAt:    time.Now(),
		}
		if res := dao.GormDB.Create(&newContact); res.Error != nil {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		anotherContact := model.UserContact{
			UserId:      contactId,
			ContactId:   ownerId,
			ContactType: contact_type_enum.User,     // 用户
			Status:      contact_status_enum.NORMAL, // 正常
			CreatedAt:   newContact.CreatedAt,
			UpdateAt:    newContact.UpdateAt,
		}
		if res := dao.GormDB.Create(&anotherContact); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		if err := cache.GetGlobalCache().DelKeyIfExists("contact_user_list_" + ownerId); err != nil {
			zlog.Error(fmt.Sprintf("删除联系人缓存失败：%v", err))
		}
		return "已添加该联系人", 0
	} else {
		var group model.GroupInfo
		if res := dao.GormDB.Where("uuid = ?", ownerId).Find(&group); res.Error != nil {
			zlog.Error(fmt.Sprintf("查询群聊信息失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		if group.Status == group_status_enum.DISABLE {
			zlog.Warn(fmt.Sprintf("群聊 %s 已被禁用, 无法添加为联系人", group.Uuid))
			return "该群聊已被禁用", -2
		}
		contactApply.Status = contact_apply_status_enum.AGREE
		if res := dao.GormDB.Save(&contactApply); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		// 群聊就只用创建一个UserContact，因为一个UserContact足以表达双方的状态
		newContact := model.UserContact{
			UserId:      contactId,
			ContactId:   ownerId,
			ContactType: contact_type_enum.Group,    // 群聊
			Status:      contact_status_enum.NORMAL, // 正常
			CreatedAt:   time.Now(),
			UpdateAt:    time.Now(),
		}
		if res := dao.GormDB.Create(&newContact); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		var members []string
		if err := json.Unmarshal(group.Members, &members); err != nil {
			zlog.Error(fmt.Sprintf("解析群聊成员失败：%v", err))
			return constants.SYSTEM_ERROR, -1
		}
		members = append(members, contactId)
		group.MemberCnt = len(members)
		group.Members, _ = json.Marshal(members)
		if res := dao.GormDB.Save(&group); res.Error != nil {
			zlog.Error(fmt.Sprintf("保存群聊成员失败：%v", res.Error))
			return constants.SYSTEM_ERROR, -1
		}
		if err := cache.GetGlobalCache().DelKeyIfExists("my_joined_group_list_" + ownerId); err != nil {
			zlog.Error(fmt.Sprintf("删除加入群聊缓存失败：%v", err))
		}
		return "已通过加群申请", 0
	}
}

// RefuseContactApply 拒绝联系人申请 -✅
func (u *userContactService) RefuseContactApply(ownerId string, contactId string) (string, int) {
	// ownerId 如果是用户的话就是登录用户，如果是群聊的话就是群聊id
	var contactApply model.ContactApply
	if res := dao.GormDB.Where("contact_id = ? AND user_id = ?", ownerId, contactId).First(&contactApply); res.Error != nil {
		zlog.Error(fmt.Sprintf("查询联系人申请记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	contactApply.Status = contact_apply_status_enum.REFUSE
	if res := dao.GormDB.Save(&contactApply); res.Error != nil {
		zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	if ownerId[0] == 'U' {
		return "已拒绝该联系人申请", 0
	} else {
		return "已拒绝该加群申请", 0
	}

}

// BlackContact 拉黑联系人	-✅
func (u *userContactService) BlackContact(ownerId string, contactId string) (string, int) {
	// 拉黑
	if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", ownerId, contactId).Updates(map[string]interface{}{
		"status":    contact_status_enum.BLACK,
		"update_at": time.Now(),
	}); res.Error != nil {
		zlog.Error(fmt.Sprintf("拉黑联系人失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	// 被拉黑
	if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", contactId, ownerId).Updates(map[string]interface{}{
		"status":    contact_status_enum.BE_BLACK,
		"update_at": time.Now(),
	}); res.Error != nil {
		zlog.Error(fmt.Sprintf("拉黑联系人失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	// 删除会话
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败：%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	resp := sessionClient.DeleteSessionsByUsers(ownerId, contactId)
	if resp.Code != 0 {
		zlog.Error(fmt.Sprintf("删除会话失败：%v", resp.Message))
		return resp.Message, int(resp.Code)
	}
	if err := cache.GetGlobalCache().DelKeyIfExists("contact_user_list_" + ownerId); err != nil {
		zlog.Error(fmt.Sprintf("删除联系人缓存失败：%v", err))
	}

	return "已拉黑该联系人", 0
}

// CancelBlackContact 取消拉黑联系人	-✅
func (u *userContactService) CancelBlackContact(ownerId string, contactId string) (string, int) {
	// 因为前端的设定，这里需要判断一下ownerId和contactId是不是有拉黑和被拉黑的状态
	var blackContact model.UserContact
	if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", ownerId, contactId).First(&blackContact); res.Error != nil {
		zlog.Error(fmt.Sprintf("查询拉黑联系人记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	if blackContact.Status != contact_status_enum.BLACK {
		return "未拉黑该联系人，无需解除拉黑", -2
	}

	// 取消拉黑
	blackContact.Status = contact_status_enum.NORMAL
	if res := dao.GormDB.Save(&blackContact); res.Error != nil {
		zlog.Error(fmt.Sprintf("取消拉黑联系人失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	if res := dao.GormDB.Save(&blackContact); res.Error != nil {
		zlog.Error(fmt.Sprintf("取消拉黑联系人失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	return "已解除拉黑该联系人", 0
}

// BlackApply 拉黑申请	-✅
func (u *userContactService) BlackApply(ownerId string, contactId string) (string, int) {
	var contactApply model.ContactApply
	if res := dao.GormDB.Where("contact_id = ? AND user_id = ?", ownerId, contactId).First(&contactApply); res.Error != nil {
		zlog.Error(fmt.Sprintf("查询联系人申请记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	contactApply.Status = contact_apply_status_enum.BLACK
	if res := dao.GormDB.Save(&contactApply); res.Error != nil {
		zlog.Error(fmt.Sprintf("保存联系人申请记录失败：%v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}
	return "已拉黑该申请", 0
}

// GetUserContact 查询好友关系 -✅
func (u *userContactService) GetUserContact(userId string, contactId string) (string, respond.GetUserContactResponse, int) {
	var contact model.UserContact
	if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).First(&contact); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return "该好友关系不存在", respond.GetUserContactResponse{}, -2
		}
		zlog.Error(fmt.Sprintf("查询好友关系失败：%v", res.Error))
		return constants.SYSTEM_ERROR, respond.GetUserContactResponse{}, -1
	}
	return "查询好友关系成功", respond.GetUserContactResponse{
		UserId:      contact.UserId,
		ContactId:   contact.ContactId,
		ContactType: contact.ContactType,
		Status:      contact.Status,
	}, 0
}