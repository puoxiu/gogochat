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

