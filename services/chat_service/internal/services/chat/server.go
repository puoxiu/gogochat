package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/puoxiu/gogochat/common/cache"
	"github.com/puoxiu/gogochat/common/clients"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/enum/message/message_status_enum"
	"github.com/puoxiu/gogochat/pkg/enum/message/message_type_enum"
	"github.com/puoxiu/gogochat/services/chat_service/internal/dao"
	"github.com/puoxiu/gogochat/services/chat_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/chat_service/internal/dto/respond"
	"github.com/puoxiu/gogochat/services/chat_service/internal/model"

	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

const (
	MsgStatusSuccess      = 0  // 发送成功
	MsgStatusServerError  = -1 // 服务端错误
	MsgStatusNotFriend    = -2 // 检查好友关系 可能被删、拉黑等
)

type Server struct {
	Clients  map[string]*Client
	mutex    *sync.Mutex
	Transmit chan []byte  // 转发通道
	Login    chan *Client // 登录通道
	Logout   chan *Client // 退出登录通道
}

var ChatServer *Server

func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients:  make(map[string]*Client),
			mutex:    &sync.Mutex{},
			Transmit: make(chan []byte, constants.CHANNEL_SIZE),
			Login:    make(chan *Client, constants.CHANNEL_SIZE),
			Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

// normalizePath 将https://127.0.0.1:8000/static/xxx 转为 /static/xxx
func normalizePath(path string) string {
	// 查找 "/static/" 的位置
	if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
		return path
	}
	staticIndex := strings.Index(path, "/static/")
	if staticIndex < 0 {
		// log.Println(path)
		zlog.Error("路径不合法")
	}
	// 返回从 "/static/" 开始的部分
	return path[staticIndex:]
}

// validateMessage 在发送之前 进行检验 准备工作
func (s *Server) validateMessage(message *model.Message) bool {
		// 判断是否是正常好友关系
	userClients, err := clients.GetGlobalUserClient()
	if err != nil {
		zlog.Error("获取用户客户端失败: " + err.Error())
		// 向发送者反馈服务端错误
		s.mutex.Lock()
		if sendClient, ok := s.Clients[message.SendId]; ok {
			sendMessageToClient(sendClient, message, MsgStatusServerError)
		}
		s.mutex.Unlock()
		return false
	}
	resp := userClients.GetContactStatus(message.SendId, message.ReceiveId)
	if resp.Code == -1 {
		zlog.Error("查询好友关系失败: " + resp.Message)
		s.mutex.Lock()
		if sendClient, ok := s.Clients[message.SendId]; ok {
			sendMessageToClient(sendClient, message, MsgStatusServerError)
		}
		s.mutex.Unlock()
		return false
	}
	if resp.Status != 0 {
		zlog.Info("用户" + message.SendId + "和用户" + message.ReceiveId + "不是好友关系")
		s.mutex.Lock()
		if sendClient, ok := s.Clients[message.SendId]; ok {
			sendMessageToClient(sendClient, message, MsgStatusNotFriend)
		}
		s.mutex.Unlock()
		return false
	}
	// 为接收者创建与发送者的会话（如果不存在）
	sessionClient, err := clients.GetGlobalSessionClient()
	if err != nil {
		zlog.Error("获取会话客户端失败: " + err.Error())
	} else {
		if resp := sessionClient.CreateSessionIfNotExist(message.ReceiveId, message.SendId); resp.Code != 0 {
			zlog.Error("为接收者创建会话失败: " + resp.Message)
		}
	}
	return true
}

func (s *Server) Start() {
	defer func() {
		close(s.Transmit)
		close(s.Logout)
		close(s.Login)
	}()
	for {
		select {
		case client := <-s.Login:
			{
				s.mutex.Lock()
				s.Clients[client.Uuid] = client
				s.mutex.Unlock()
				zlog.Debug(fmt.Sprintf("欢迎来到gogo聊天服务器,亲爱的用户%s\n", client.Uuid))
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte("欢迎来到gogo聊天服务器"))
				if err != nil {
					zlog.Error(err.Error())
				}
			}

		case client := <-s.Logout:
			{
				s.mutex.Lock()
				delete(s.Clients, client.Uuid)
				s.mutex.Unlock()
				zlog.Info(fmt.Sprintf("用户%s退出登录\n", client.Uuid))
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("已退出登录")); err != nil {
					zlog.Error(err.Error())
				}
				// log.Printf("检测到用户退出啦 ：%s logout", client.Uuid)
			}

		case data := <-s.Transmit:
			{
				var chatMessageReq request.ChatMessageRequest
				if err := json.Unmarshal(data, &chatMessageReq); err != nil {
					zlog.Error(err.Error())
				}
				if chatMessageReq.Type == message_type_enum.Text {
					// 存message
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    chatMessageReq.Content,
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "0B",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入 避免后续服务部署 IP 变更导致头像加载失败。
					// 例如：https://127.0.0.1:8000/static/xxx 转为 /static/xxx
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' {
						if !s.validateMessage(&message) {
							continue
						}

						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: message.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							sendMessageToClient(receiveClient, &message, MsgStatusSuccess)
						}
						if sendClient, ok := s.Clients[message.SendId]; ok {
							sendMessageToClient(sendClient, &message, MsgStatusSuccess)
						}
						s.mutex.Unlock()

						if rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId); err == nil {
							var rsp []respond.GetMessageListRespond
							if err = json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							// 将当前消息追加到缓存列表中，并且序列化为字符串存储到Redis中
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := cache.GetGlobalCache().SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					} else if message.ReceiveId[0] == 'G' {
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						// log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err = json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							// 遍历发送 并且排除发送者
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								// 发送给自己
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = cache.GetGlobalCache().GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := cache.GetGlobalCache().SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.File {
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        chatMessageReq.Url,
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   chatMessageReq.FileSize,
						FileType:   chatMessageReq.FileType,
						FileName:   chatMessageReq.FileName,
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' {
						if !s.validateMessage(&message) {
							fmt.Println("验证不通过")
							continue
						}
						fmt.Println("验证通过")

						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}

						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							sendMessageToClient(receiveClient, &message, MsgStatusSuccess)
						}
						if sendClient, ok := s.Clients[message.SendId]; ok {
							sendMessageToClient(sendClient, &message, MsgStatusSuccess)
						}
						s.mutex.Unlock()

						if rspString, err := cache.GetGlobalCache().GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId); err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := cache.GetGlobalCache().SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					} else {
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						// log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = cache.GetGlobalCache().GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err = json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := cache.GetGlobalCache().SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
					fmt.Println("到这里")
				} else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
					var avData request.AVData	//  音视频信令结构体（含通话类型、通话ID等）
					if err := json.Unmarshal([]byte(chatMessageReq.AVdata), &avData); err != nil {
						zlog.Error(err.Error())
					}
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     chatMessageReq.AVdata,
					}
					if avData.MessageId == "PROXY" && (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
						// 存message
						// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
						message.SendAvatar = normalizePath(message.SendAvatar)
						if res := dao.GormDB.Create(&message); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
					}

					if chatMessageReq.ReceiveId[0] == 'U' {
						if !s.validateMessage(&message) {
							continue
						}

						messageRsp := respond.AVMessageRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: message.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
							AVdata:     message.AVdata,
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						// log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						// log.Println("返回的消息为：", messageRsp)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 通话这不能回显，发回去的话就会出现两个start_call。
						//sendClient := s.Clients[message.SendId]
						//sendClient.SendBack <- messageBack
						s.mutex.Unlock()
					}
				}

			}
		}
	}
}


func sendMessageToClient(client *Client, message *model.Message, code int8) {
    // 基础响应体
    messageRsp := respond.GetMessageListRespond{
        SendId:     message.SendId,
        SendName:   message.SendName,
        SendAvatar: message.SendAvatar,
        ReceiveId:  message.ReceiveId,
        Type:       message.Type,
        Url:        message.Url,
        FileSize:   message.FileSize,
        FileName:   message.FileName,
        FileType:   message.FileType,
        CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"), // 补充时间
    }

    // 根据状态码设置提示内容
    switch code {
    case MsgStatusServerError:
        messageRsp.Content = "系统消息：消息发送失败（服务端错误）"
    case MsgStatusNotFriend:
        messageRsp.Content = "系统消息：消息发送失败，请检查好友关系"
    default:
        messageRsp.Content = message.Content // 正常消息用原内容
    }

    // 序列化并发送
    jsonMessage, err := json.Marshal(messageRsp)
    if err != nil {
        zlog.Error("消息序列化失败: " + err.Error())
        return
    }
    messageBack := &MessageBack{
        Message: jsonMessage,
        Uuid:    message.Uuid,
    }

    // 非阻塞发送，避免通道阻塞导致的问题
    select {
    case client.SendBack <- messageBack:
        zlog.Info("消息已发送到客户端: " + client.Uuid)
    default:
        zlog.Warn("客户端通道已满，消息发送失败: " + client.Uuid)
    }
}




func (s *Server) Close() {
	close(s.Login)
	close(s.Logout)
	close(s.Transmit)
}

func (s *Server) GetClient(id string) (*Client, bool) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    c, ok := s.Clients[id]
    return c, ok
}

func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	s.Logout <- client
	s.mutex.Unlock()
}

func (s *Server) SendMessageToTransmit(message []byte) {
	s.mutex.Lock()
	s.Transmit <- message
	s.mutex.Unlock()
}

func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}
