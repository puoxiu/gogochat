package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/enum/contact/contact_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/contact/contact_type_enum"
	"github.com/puoxiu/gogochat/pkg/enum/group_info/group_status_enum"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/user_service/internal/dao"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/user_service/internal/dto/respond"
	"github.com/puoxiu/gogochat/services/user_service/internal/model"
	"gorm.io/gorm"

	"time"
)

type groupInfoService struct {
}
var GroupInfoService = new(groupInfoService)

// CreateGroup 创建群聊
func (g *groupInfoService) CreateGroup(groupReq request.CreateGroupRequest) (string, int) {
    tx := dao.GormDB.Begin()
    if tx.Error != nil {
        zlog.Error(fmt.Sprintf("开启事务失败: %v", tx.Error))
        return constants.SYSTEM_ERROR, -1
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }() 
	group := model.GroupInfo{
		Uuid:      fmt.Sprintf("G%s", random.GetNowAndLenRandomString(11)),
		Name:      groupReq.Name,
		Notice:    groupReq.Notice,
		OwnerId:   groupReq.OwnerId,
		MemberCnt: 1,
		AddMode:   groupReq.AddMode,
		Avatar:    groupReq.Avatar,
		Status:    group_status_enum.NORMAL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	var members []string
	members = append(members, groupReq.OwnerId)
	var err error
	group.Members, err = json.Marshal(members)
	if err != nil {
		zlog.Error(fmt.Sprintf("序列化群组成员失败: %v", err))
		tx.Rollback()
		return constants.SYSTEM_ERROR, -1
	}
	if res := tx.Create(&group); res.Error != nil {
		zlog.Error(fmt.Sprintf("创建群聊失败: %v", res.Error))
		tx.Rollback()
		return constants.SYSTEM_ERROR, -1
	}

	// 添加联系人
	contact := model.UserContact{
		UserId:      groupReq.OwnerId,
		ContactId:   group.Uuid,
		ContactType: contact_type_enum.Group,
		Status:      contact_status_enum.NORMAL,
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}
	if res := tx.Create(&contact); res.Error != nil {
		zlog.Error(fmt.Sprintf("创建联系人失败: %v", res.Error))
		tx.Rollback()
		return constants.SYSTEM_ERROR, -1
	}
    if res := tx.Commit(); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("事务提交失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

	if err := cache.GetGlobalCache().DelKeysWithPattern("contact_mygroup_list_" + groupReq.OwnerId); err != nil {
		zlog.Error(fmt.Sprintf("删除缓存失败:%s", err.Error()))
	}

	return "创建成功", 0
}

// LoadMyGroup 获取我创建的群聊
func (g *groupInfoService) LoadMyGroup(ownerId string) (string, []respond.LoadMyGroupRespond, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("contact_mygroup_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var groupList []model.GroupInfo
			if res := dao.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			if len(groupList) == 0 {
				// 即使结果为空，也写入缓存(避免缓存穿透)
				emptyRsp := []respond.LoadMyGroupRespond{}
				emptyRspString, err := json.Marshal(emptyRsp)
				if err != nil {
					zlog.Error(fmt.Sprintf("序列化失败:%s", err.Error()))
					return constants.SYSTEM_ERROR, nil, -1
				}
				if err := cache.GetGlobalCache().SetKeyEx("contact_mygroup_list_"+ownerId, string(emptyRspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
					zlog.Error(err.Error())
				}
				return "您还没有创建任何群聊", nil, -2
			}
			var groupListRsp []respond.LoadMyGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, respond.LoadMyGroupRespond{
					GroupId:   group.Uuid,
					GroupName: group.Name,
					Avatar:    group.Avatar,
				})
			}
			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zlog.Error(err.Error())
			}
			if err := cache.GetGlobalCache().SetKeyEx("contact_mygroup_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Warn(err.Error())
			}
			return "获取成功", groupListRsp, 0
		} else {
			zlog.Error(fmt.Sprintf("获取缓存失败:%s", err.Error()))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var groupListRsp []respond.LoadMyGroupRespond
	if err := json.Unmarshal([]byte(rspString), &groupListRsp); err != nil {
		zlog.Error(fmt.Sprintf("解析缓存失败:%s", err.Error()))
		return constants.SYSTEM_ERROR, nil, -1
	}
	return "获取成功", groupListRsp, 0
}

// GetGroupInfo 获取群聊详情
func (g *groupInfoService) GetGroupInfo(groupId string) (string, *respond.GetGroupInfoRespond, int) {
	cacheKey := "group_info_" + groupId
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr(cacheKey)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zlog.Error(fmt.Sprintf("groupId = %s", groupId))
					return constants.SYSTEM_ERROR, nil, -2
				} else {
					zlog.Error(fmt.Sprintf("查询数据库失败:%s", res.Error.Error()))
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			rsp := &respond.GetGroupInfoRespond{
				Uuid:      group.Uuid,
				Name:      group.Name,
				Notice:    group.Notice,
				Avatar:    group.Avatar,
				MemberCnt: group.MemberCnt,
				OwnerId:   group.OwnerId,
				AddMode:   group.AddMode,
				Status:    group.Status,
				IsDeleted: group.DeletedAt.Valid,
			}
			rspString, err := json.Marshal(rsp)
			if err != nil {
				zlog.Error(err.Error())
			} 
			if err := cache.GetGlobalCache().SetKeyEx(cacheKey, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(fmt.Sprintf("设置缓存失败:%s", err.Error()))
			}
			return "获取成功", rsp, 0
		} else {
			zlog.Error(fmt.Sprintf("获取缓存失败:%s", err.Error()))
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp *respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(fmt.Sprintf("解析缓存失败:%s", err.Error()))
		return constants.SYSTEM_ERROR, nil, -1
	}
	return "获取成功", rsp, 0
}



// LeaveGroup 退群
func (g *groupInfoService) LeaveGroup(userId string, groupId string) (string, int) {
	tx := dao.GormDB.Begin()
	if tx.Error != nil {
		zlog.Error(tx.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var group model.GroupInfo
	if res := tx.Set("gorm:query_option", "FOR UPDATE").First(&group, "uuid = ?", groupId); res.Error != nil {
		tx.Rollback()
        if errors.Is(res.Error, gorm.ErrRecordNotFound) {
            return "群聊不存在", -2
        }
		zlog.Error((fmt.Sprintf("查询群聊失败: %v", res.Error)))
		return constants.SYSTEM_ERROR, -1
	}
		
	// 检查用户是否在群聊中
	var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        tx.Rollback()
        zlog.Error(err.Error())
        return constants.SYSTEM_ERROR, -1
    }
	isInGroup := false
	for _, member := range members {
		if member == userId {
			isInGroup = true
			break
		}
	}
	if !isInGroup {
		tx.Rollback()
		return "用户不在群聊中", -2
	}

	// 若是群主，不能退群
	if group.OwnerId == userId {
		tx.Rollback()
		return "群主不能退群", -2
	}

	newMembers := make([]string, 0, len(members)-1)
	for _, member := range members {
		if member != userId {
			newMembers = append(newMembers, member)
		}
	}
	membersJson, err := json.Marshal(newMembers)
	if err != nil {
		tx.Rollback()
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	group.Members = membersJson
	group.MemberCnt = len(newMembers)
	if res := tx.Save(&group); res.Error != nil {
		tx.Rollback()
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	
	// 软删除会话--调用rpc
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败:%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	// 删除用户userId 与群聊groupId 的会话
	resp := sessionClient.DeleteSessionsByUsers(userId, groupId)
	if resp.Code != 0 {
		tx.Rollback()
		zlog.Warn(fmt.Sprintf("删除会话失败:%v", resp.Message))
		return resp.Message, int(resp.Code)
	}
	
	// 软删除联系人
	deletedAt := gorm.DeletedAt{Time: time.Now(), Valid: true}
	if res := tx.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, groupId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     contact_status_enum.QUIT_GROUP, // 退群
	}); res.Error != nil {
		tx.Rollback()
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	
	// 删除申请记录
	if res := tx.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", groupId, userId).Update("deleted_at", deletedAt); res.Error != nil {
		tx.Rollback()
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 提交事务
    if res := tx.Commit(); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

	// 从缓存中删除群聊信息
    cacheKeys := []string{
        "group_info_" + groupId,
        "group_memberlist_" + groupId,
        "group_session_list_" + userId,
        "my_joined_group_list_" + userId, // 注意原代码多了个空格，需修正
    }
	for _, key := range cacheKeys {
        if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
            zlog.Warn(fmt.Sprintf("清理缓存失败: key=%s, err=%s", key, err.Error()))
        }
    }

	return "退群成功", 0
}

// DismissGroup 解散群聊
func (g *groupInfoService) DismissGroup(ownerId, groupId string) (string, int) {
	// 开启数据库事务:确保“解散群聊+删除关联数据”原子性(要么全成功，要么全失败)
	tx := dao.GormDB.Begin()
	if tx.Error != nil {
		zlog.Error(fmt.Sprintf("开启事务失败: %v", tx.Error))
		return constants.SYSTEM_ERROR, -1
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var group model.GroupInfo
	if res := tx.Set("gorm:query_option", "FOR UPDATE").First(&group, "uuid = ?", groupId); res.Error != nil {
		tx.Rollback()
        if errors.Is(res.Error, gorm.ErrRecordNotFound) {
            return "群聊不存在或已被解散", -2
        }
		zlog.Error((fmt.Sprintf("查询群聊失败: %v", res.Error)))
		return constants.SYSTEM_ERROR, -1
	}

	// 检查用户是否是群主
	if group.OwnerId != ownerId {
		tx.Rollback()
		return "非群主不能解散群聊", -2
	}
	// 群聊已解散则无需重复操作
	if group.DeletedAt.Valid {
		tx.Rollback()
		return "群聊已解散，无需重复操作", -2
	}

    deletedAt := gorm.DeletedAt{Time:  time.Now(), Valid: true}
	if res := tx.Model(&group).Updates(
		map[string]interface{}{
			"deleted_at": deletedAt,
			"updated_at": deletedAt.Time,
			"status":     group_status_enum.DISSOLVE, // 解散
		}); res.Error != nil {
		tx.Rollback()
		zlog.Error(fmt.Sprintf("软删除群聊失败: %v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}

	// 删除会话 和联系人
    var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("解析群成员列表失败: groupId=%s, err=%v", groupId, err))
        return constants.SYSTEM_ERROR, -1
    }

	// // 软删除群成员的联系人记录（UserContact）
    if res := tx.Model(&model.UserContact{}).
        Where("contact_id = ? AND contact_type = 1", groupId).
        Updates(map[string]interface{}{
			"deleted_at": deletedAt,
			"status":     contact_status_enum.BE_DELETE,
		}); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("删除群成员联系人记录失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

	// // 软删除群聊相关的会话记录（Session）
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		tx.Rollback()
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败:%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	for _, member := range members {
		resp := sessionClient.DeleteSessionsByUsers(member, groupId)
		if resp.Code == -2 {
			zlog.Warn(fmt.Sprintf("不存在会话记录, userId=%s, groupId=%s", member, groupId))
			continue
		}
		if resp.Code != 0 {
			tx.Rollback()
			zlog.Warn(fmt.Sprintf("rpc调用 删除会话失败:%v", resp.Message))
			return resp.Message, int(resp.Code)
		}
	}

	// 删除申请记录
	if res := tx.Model(&model.ContactApply{}).Where("contact_id = ?", groupId).Update("deleted_at", deletedAt); res.Error != nil {
		tx.Rollback()
		zlog.Error(fmt.Sprintf("批量删除群聊申请记录失败: %v", res.Error))
		return constants.SYSTEM_ERROR, -1
	}

	// 提交事务
    if res := tx.Commit(); res.Error != nil {
        zlog.Error(fmt.Sprintf("事务提交失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

    // 清理所有相关缓存
    baseCacheKeys := []string{
        "group_info_" + groupId,
        "group_memberlist_" + groupId,
    }
    memberCacheTemplates := []string{
        "contact_mygroup_list_%s",
        "group_session_list_%s",
        "my_joined_group_list_%s",
    }
    allCacheKeys := append([]string{}, baseCacheKeys...)
    for _, memberId := range members {
        for _, template := range memberCacheTemplates {
            allCacheKeys = append(allCacheKeys, fmt.Sprintf(template, memberId))
        }
    }
    for _, key := range allCacheKeys {
        if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
            zlog.Warn(fmt.Sprintf("清理缓存失败: key=%s, err=%s", key, err.Error()))
        }
    }

	return "群聊解散成功", 0
}

// CheckGroupAddMode 检查群聊加群方式
func (g *groupInfoService) CheckGroupAddMode(groupId string) (string, int8, int) {
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("group_info_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1, -1
			}
			return "加群方式获取成功", group.AddMode, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1, -1
		}
	}
	var rsp respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1, -1
	}
	return "加群方式获取成功", rsp.AddMode, 0
}

// EnterGroupDirectly 直接进群
// groupId 是群聊id 
// userId 是用户id
// EnterGroupDirectly 直接进群(无需申请，直接加入)
// 参数说明:
// - groupId:目标群聊ID(原参数名ownerId命名错误，修正为groupId)
// - userId:进群用户ID(原参数名contactId命名错误，修正为userId)
func (g *groupInfoService) EnterGroupDirectly(groupId, userId string) (string, int) {
    tx := dao.GormDB.Begin()
    if tx.Error != nil {
        zlog.Error(fmt.Sprintf("开启事务失败: %v", tx.Error))
        return constants.SYSTEM_ERROR, -1
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            zlog.Error(fmt.Sprintf("事务panic回滚: %v", r))
        }
    }()

    // 查询群聊信息(加行锁，防并发修改；校验群聊状态)
    var group model.GroupInfo
    // 加FOR UPDATE行锁:避免同时进群导致成员列表数据覆盖
    res := tx.Set("gorm:query_option", "FOR UPDATE").First(&group, "uuid = ?", groupId)
    if res.Error != nil {
        tx.Rollback()
        if errors.Is(res.Error, gorm.ErrRecordNotFound) {
            return "群聊不存在或已解散", -2 // 业务错误:群不存在
        }
        zlog.Error(fmt.Sprintf("查询群聊失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1 // 系统错误
    }
    if group.DeletedAt.Valid {
        tx.Rollback()
        return "群聊已解散，无法加入", -2 
    }

    var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("解析群成员失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }
    for _, member := range members {
        if member == userId {
            tx.Rollback()
            return "用户已在群聊中，无需重复加入", -2
        }
    }

    members = append(members, userId)
    membersJson, err := json.Marshal(members)
    if err != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("序列化群成员失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }
    group.Members = membersJson
    group.MemberCnt += 1 // 成员数量+1
    group.UpdatedAt = time.Now() // 更新时间戳
    if res := tx.Save(&group); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("更新群成员失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

    var existContact model.UserContact
    res = tx.First(&existContact, "user_id = ? AND contact_id = ? AND deleted_at IS NULL", userId, groupId)
    if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("查询联系人记录失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }
    if errors.Is(res.Error, gorm.ErrRecordNotFound) {
        newContact := model.UserContact{
            UserId:      userId,
            ContactId:   groupId,
            ContactType: contact_type_enum.Group,
            Status:      contact_status_enum.NORMAL,
            CreatedAt:   time.Now(),
            UpdateAt:   time.Now(),
        }
        if res := tx.Create(&newContact); res.Error != nil {
            tx.Rollback()
            zlog.Error(fmt.Sprintf("创建联系人记录失败: %v", res.Error))
            return constants.SYSTEM_ERROR, -1
        }
    }

    if res := tx.Commit(); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("事务提交失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }
    cacheKeys := []string{
        "group_info_" + groupId,
        "group_memberlist_" + groupId,
        "my_joined_group_list_" + userId, 
    }
    for _, key := range cacheKeys {
        if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
            zlog.Warn(fmt.Sprintf("清理缓存失败: key=%s, err=%v", key, err))
        }
    }

    return "进群成功", 0
}

// UpdateGroupInfo 更新群聊信息(仅群主可操作)
func (g *groupInfoService) UpdateGroupInfo(req request.UpdateGroupInfoRequest) (string, int) {
    tx := dao.GormDB.Begin()
    if tx.Error != nil {
        zlog.Error(fmt.Sprintf("开启事务失败: %v", tx.Error))
        return constants.SYSTEM_ERROR, -1
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            zlog.Error(fmt.Sprintf("事务panic回滚: %v", r))
        }
    }()

    var group model.GroupInfo
    res := tx.Set("gorm:query_option", "FOR UPDATE").First(&group, "uuid = ?", req.Uuid)
    if res.Error != nil {
        tx.Rollback()
        if errors.Is(res.Error, gorm.ErrRecordNotFound) {
            return "群聊不存在或已解散", -2
        }
        zlog.Error(fmt.Sprintf("查询群聊失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }
    if group.OwnerId != req.OwnerId {
        tx.Rollback()
        return "无权限修改群聊信息(仅群主可操作)", -2
    }
    if group.DeletedAt.Valid {
        tx.Rollback()
        return "群聊已解散，无法修改信息", -2
    }

    updateFlag := false
    if req.Name != "" && req.Name != group.Name {
        group.Name = req.Name
        updateFlag = true
    }
    if req.AddMode != -1 && req.AddMode != group.AddMode {
        group.AddMode = req.AddMode
        updateFlag = true
    }
    if req.Notice != "" && req.Notice != group.Notice {
        group.Notice = req.Notice
        updateFlag = true
    }
    if req.Avatar != "" && req.Avatar != group.Avatar {
        group.Avatar = req.Avatar
        updateFlag = true
    }
    // 无实际变更时，直接返回成功(避免无效数据库操作)
    if !updateFlag {
        tx.Rollback()
        return "未检测到需更新的信息", 0
    }
    group.UpdatedAt = time.Now()
    if res := tx.Save(&group); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("更新群聊信息失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

    if err := tx.Commit().Error; err != nil {
        zlog.Error(fmt.Sprintf("事务提交失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }

    cacheKeys := []string{
        "group_info_" + req.Uuid,          // 群聊基础信息缓存(必须清理)
        "group_memberlist_" + req.Uuid,    // 群成员列表缓存(若群名/头像在列表中展示，需清理)
        "contact_mygroup_list_" + req.OwnerId, // 群主的“我的群聊”列表(若展示群名/头像，需清理)
    }
    for _, key := range cacheKeys {
        if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
            zlog.Warn(fmt.Sprintf("清理缓存失败: key=%s, err=%v", key, err))
        }
    }

    return "群聊信息更新成功", 0
}

// GetGroupMemberList 获取群聊成员列表
func (g *groupInfoService) GetGroupMemberList(groupId string) (string, []respond.GetGroupMemberListRespond, int) {
	cacheKey := "group_memberlist_" + groupId
	rspString, err := cache.GetGlobalCache().GetKeyNilIsErr(cacheKey)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					return "群聊不存在或已解散", nil, -2
				}
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			if group.DeletedAt.Valid {
				return "群聊已解散，无法获取成员列表", nil, -2
			}
			var members []string
			if err := json.Unmarshal(group.Members, &members); err != nil {
				zlog.Error(fmt.Sprintf("解析群聊成员列表失败: %v", err))
				return constants.SYSTEM_ERROR, nil, -1
			}
		    if len(members) == 0 {
     		   return "群聊暂无成员", []respond.GetGroupMemberListRespond{}, 0
    		}
			
			type userEssential struct {
				Uuid     string `gorm:"column:uuid"`
				Nickname string `gorm:"column:nickname"`
				Avatar   string `gorm:"column:avatar"`
			}
			var essentialList []userEssential
			// 用SELECT指定字段，避免查询冗余信息；WHERE IN批量查询，减少数据库请求
			if res := dao.GormDB.Model(&model.UserInfo{}).
				Select("uuid, nickname, avatar").
				Where("uuid IN ?", members).
				Find(&essentialList); res.Error != nil {
				zlog.Error(fmt.Sprintf("批量查询群成员信息失败: %v", res.Error))
				return constants.SYSTEM_ERROR, nil, -1
			}
			// 用map快速查询，避免线性查找
			userMap := make(map[string]userEssential, len(essentialList))
			for _, ess := range essentialList {
				userMap[ess.Uuid] = ess
			}
			var rspList []respond.GetGroupMemberListRespond
			for _, memberId := range members {
				// 处理成员ID存在但用户信息不存在的异常(如用户已注销)
				ess, exists := userMap[memberId]
				if !exists {
					zlog.Warn(fmt.Sprintf("群成员[%s]信息不存在，已跳过", memberId))
					continue
				}
				// 补充“是否为群主”标识(从群信息中直接判断，无需额外查询)
				rspList = append(rspList, respond.GetGroupMemberListRespond{
					UserId:   ess.Uuid,
					Nickname: ess.Nickname,
					Avatar:   ess.Avatar,
				})
			}
			
			if len(rspList) > 0 {
				cacheData, err := json.Marshal(rspList)
				if err != nil {
					zlog.Error(fmt.Sprintf("序列化群聊成员列表缓存失败: %v", err))
				} else {
					if err := cache.GetGlobalCache().SetKeyEx(cacheKey, string(cacheData), time.Minute*constants.REDIS_TIMEOUT); err != nil {
						zlog.Warn(fmt.Sprintf("写入群成员缓存失败: %v", err))
					}
				}
			}
			return "获取群聊成员列表成功", rspList, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.GetGroupMemberListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(fmt.Sprintf("解析群聊成员列表缓存失败: %v", err))
		return constants.SYSTEM_ERROR, nil, -1
	}
	return "获取群聊成员列表成功", rsp, 0
}

// RemoveGroupMembers 移除群聊成员(仅群主可操作，支持批量移除)
func (g *groupInfoService) RemoveGroupMembers(req request.RemoveGroupMembersRequest) (string, int) {
    tx := dao.GormDB.Begin()
    if tx.Error != nil {
        zlog.Error(fmt.Sprintf("开启事务失败: %v", tx.Error))
        return constants.SYSTEM_ERROR, -1
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            zlog.Error(fmt.Sprintf("事务panic回滚: %v", r))
        }
    }()

    var group model.GroupInfo
    res := tx.Set("gorm:query_option", "FOR UPDATE").First(&group, "uuid = ?", req.GroupId)
    if res.Error != nil {
        tx.Rollback()
        if errors.Is(res.Error, gorm.ErrRecordNotFound) {
            return "群聊不存在或已解散", -2
        }
        zlog.Error(fmt.Sprintf("查询群聊失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }
    if group.OwnerId != req.OwnerId {
        tx.Rollback()
        return "无权限移除群成员(仅群主可操作)", -2
    }
    if group.DeletedAt.Valid {
        tx.Rollback()
        return "群聊已解散，无法移除成员", -2
    }

    var currentMembers []string
    if err := json.Unmarshal(group.Members, &currentMembers); err != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("解析群成员列表失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }
    // 4.1 用map存储当前成员，O(1)判断是否在群内(优化循环效率)
    currentMemberMap := make(map[string]bool, len(currentMembers))
    for _, member := range currentMembers {
        currentMemberMap[member] = true
    }
    toRemoveSet := make(map[string]bool, len(req.UuidList))
    for _, uuid := range req.UuidList {
        if uuid == req.OwnerId {
            tx.Rollback()
            return fmt.Sprintf("不能移除群主(成员ID:%s)", uuid), -6
        }
        if !currentMemberMap[uuid] {
            tx.Rollback()
            return fmt.Sprintf("成员(ID:%s)不在群内，无需移除", uuid), -7
        }
        toRemoveSet[uuid] = true
    }

    var newMembers []string
    for _, member := range currentMembers {
        if !toRemoveSet[member] { 
            newMembers = append(newMembers, member)
        }
    }
    if len(newMembers) != len(currentMembers)-len(toRemoveSet) {
        tx.Rollback()
        zlog.Error("群成员数量计算错误，移除操作终止")
        return constants.SYSTEM_ERROR, -1
    }

    deletedAt := gorm.DeletedAt{Time: time.Now(), Valid: true}
    toRemoveList := make([]string, 0, len(toRemoveSet))
    for uuid := range toRemoveSet {
        toRemoveList = append(toRemoveList, uuid)
    }
	// 删除对应成员的会话 调用rpc
		// 软删除会话--调用rpc
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error(fmt.Sprintf("获取会话服务客户端失败:%v", err))
		return constants.SYSTEM_ERROR, -1
	}
	// 删除用户userId 与群聊groupId 的会话
	for uuid := range toRemoveSet {
		resp := sessionClient.DeleteSessionsByUsers(uuid, req.GroupId)
		if resp.Code == -2 {
			zlog.Warn(fmt.Sprintf("用户[%s]与群聊[%s]会话不存在，无需删除", uuid, req.GroupId))
            continue // 跳过不存在的会话，继续执行后续逻辑
		}
		if resp.Code != 0 {
			tx.Rollback()
			zlog.Error(fmt.Sprintf("rpc 删除用户[%s]与群聊[%s]会话失败: %s", uuid, req.GroupId, resp.Message))
			return constants.SYSTEM_ERROR, -1
		}
	}

    // 批量删除待移除成员的联系人记录
    if res := tx.Model(&model.UserContact{}).
        Where("user_id IN ? AND contact_id = ?", toRemoveList, req.GroupId).
        Update("deleted_at", deletedAt); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("批量删除成员联系人失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }
    if res := tx.Model(&model.ContactApply{}).
        Where("user_id IN ? AND contact_id = ?", toRemoveList, req.GroupId).
        Update("deleted_at", deletedAt); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("批量删除成员申请记录失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

    newMembersJson, err := json.Marshal(newMembers)
    if err != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("序列化新成员列表失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }
    group.Members = newMembersJson
    group.MemberCnt = len(newMembers) 
    group.UpdatedAt = time.Now()
    if res := tx.Save(&group); res.Error != nil {
        tx.Rollback()
        zlog.Error(fmt.Sprintf("更新群聊成员列表失败: %v", res.Error))
        return constants.SYSTEM_ERROR, -1
    }

    if err := tx.Commit().Error; err != nil {
        zlog.Error(fmt.Sprintf("事务提交失败: %v", err))
        return constants.SYSTEM_ERROR, -1
    }

    cacheKeys := []string{
        "group_info_" + req.GroupId,          // 群聊基础信息缓存(必须清理)
        "group_memberlist_" + req.GroupId,    // 群成员列表缓存(必须清理)
        "contact_mygroup_list_" + req.OwnerId, // 群主的“我的群聊”列表(若展示成员数，需清理)
    }
    // 补充:清理被移除成员的“已加入群聊”列表缓存
    for _, uuid := range toRemoveList {
        cacheKeys = append(cacheKeys, "my_joined_group_list_"+uuid)
        cacheKeys = append(cacheKeys, "group_session_list_"+uuid) // 被移除成员的群会话列表
    }
    for _, key := range cacheKeys {
        if err := cache.GetGlobalCache().DelKeyIfExists(key); err != nil {
            zlog.Warn(fmt.Sprintf("清理缓存失败: key=%s, err=%v", key, err))
        }
    }

    return fmt.Sprintf("成功移除%d名群成员", len(toRemoveList)), 0
}
