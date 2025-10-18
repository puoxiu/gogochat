## 缓存一致性策略
先更新库 再删除缓存


## 缓存键值
1. 短信验证码缓存键值：
key : auth_code_<phone_number>
value : <6位短信验证码>
有效时间 : 1分钟

2. 用户信息缓存键值：
key : user_info_<uuid>
value : <用户信息>
有效时间 : 1小时

3. 联系人列表缓存键值：
key : contact_user_list_<uuid>
value : <联系人列表>
有效时间 : 1小时

4. 加入群聊列表缓存键值：
key : my_joined_group_list_<uuid>
value : <加入群聊列表>
有效时间 : 1小时

5. 会话缓存键值：
key : session_<send_id>_<receive_id>
value : <会话信息>
有效时间 : 1小时

6. 会话列表缓存键值：
key : session_list_<uuid>
value : <会话列表>
有效时间 : 1小时

7. 群聊会话列表缓存键值：
key : group_session_list_<uuid>
value : <群聊会话列表>
有效时间 : 1小时

8. 我创建的群聊列表缓存键值：
key : contact_mygroup_list_<uuid>
value : <我创建的群聊列表>
有效时间 : 1小时

9. 群聊详细信息缓存键值：
key: group_info_<groupId>
value: <群聊详细信息: 群聊ID, 群聊名称, 群聊描述, 群聊创建时间, 群聊成员列表, 群聊更新时间, 群聊删除时间等>
有效时间: 1小时

10. 群聊成员列表缓存键值：
key : group_memberlist_<groupId>
value : <群聊成员列表: 用户ID, 昵称, 头像>
有效时间 : 1小时 
