package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	mykafka "github.com/puoxiu/gogochat/common/kafka"
	"github.com/puoxiu/gogochat/pkg/constants"
	"github.com/puoxiu/gogochat/pkg/enum/message/message_status_enum"
	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/puoxiu/gogochat/services/chat_service/internal/config"
	"github.com/puoxiu/gogochat/services/chat_service/internal/dao"
	"github.com/puoxiu/gogochat/services/chat_service/internal/dto/request"
	"github.com/puoxiu/gogochat/services/chat_service/internal/model"
	"github.com/segmentio/kafka-go"
)

type MessageBack struct {
	Message []byte
	Uuid    string
}

type Client struct {
	Conn     *websocket.Conn
	Uuid     string	
	SendTo   chan []byte       // 给server端
	SendBack chan *MessageBack // 给前端
	HeartBeatDone     chan struct{}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	// 检查连接的Origin头
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var ctx = context.Background()

// 读取websocket消息并发送给send通道
func (c *Client) Read() {
	zlog.Info("ws read goroutine start")
	defer func() {
	    if r := recover(); r != nil {
        	zlog.Error(fmt.Sprintf("panic in Read(): %v", r))
    	}
		ClientLogout(c.Uuid)
	}()

	for {
		_, jsonMessage, err := c.Conn.ReadMessage()
		if err != nil {
			zlog.Error(err.Error())
			return
		} else {
			var message = request.ChatMessageRequest{}
			if err := json.Unmarshal(jsonMessage, &message); err != nil {
				zlog.Error(err.Error())
			}
			if config.AppConfig.KafkaConfig.MessageMode == "channel" {
				// 如果server的转发channel没满，先把sendto中的给transmit
				for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
					sendToMessage := <-c.SendTo
					ChatServer.SendMessageToTransmit(sendToMessage)
				}
				// 如果server没满，sendto空了，直接给server的transmit
				if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
					ChatServer.SendMessageToTransmit(jsonMessage)
				} else if len(c.SendTo) < constants.CHANNEL_SIZE {
					// 如果server满了，直接塞sendto
					c.SendTo <- jsonMessage
				} else {
					// 否则考虑加宽channel size，或者使用kafka
					if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("由于目前同一时间过多用户发送消息，消息发送失败，请稍后重试")); err != nil {
						zlog.Error(err.Error())
					}
				}
			} else {
				if err := mykafka.KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
					Key:   []byte(strconv.Itoa(config.AppConfig.KafkaConfig.Partition)),
					Value: jsonMessage,
				}); err != nil {
					zlog.Error(err.Error())
				}
				zlog.Info("已发送消息：" + string(jsonMessage))
			}
		}
	}
}

// 从send通道读取消息发送给websocket
func (c *Client) Write() {
	zlog.Info("ws write goroutine start")
	for messageBack := range c.SendBack { // 阻塞状态
		// 通过 WebSocket 发送消息
		err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		// 说明顺利发送，修改状态为已发送
		if res := dao.GormDB.Model(&model.Message{}).Where("uuid = ?", messageBack.Uuid).Update("status", message_status_enum.Sent); res.Error != nil {
			zlog.Error(res.Error.Error())
		}
	}
}

// NewClientInit 当接受到前端有登录消息时，会调用该函数
func NewClientInit(c *gin.Context, clientId string) {
	kafkaConfig := config.AppConfig.KafkaConfig
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Error(err.Error())
	}
	client := &Client{
		Conn:     conn,
		Uuid:     clientId,
		SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
		SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
		HeartBeatDone: make(chan struct{}),
	}
	if kafkaConfig.MessageMode == "channel" {
		ChatServer.SendClientToLogin(client)
	} else {
		KafkaChatServer.SendClientToLogin(client)
	}
	go client.Read()
	go client.Write()
	zlog.Info("ws连接成功")

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})
	go client.Heartbeat()
}

// Heartbeat 启动定时发送Ping的协程
func (c *Client) Heartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				ClientLogout(c.Uuid)
				return
			}
		case <-c.HeartBeatDone:
			return
		}
	}
}



// ClientLogout 当接受到前端有登出消息时，会调用该函数
func ClientLogout(clientId string) (string, int) {
	kafkaConfig := config.AppConfig.KafkaConfig
	var (
		client *Client
		ok     bool
	)
	if kafkaConfig.MessageMode == "channel" {
		client, ok = ChatServer.GetClient(clientId)
	} else {
		client, ok = KafkaChatServer.GetClient(clientId)
	}
	if !ok || client == nil {
		zlog.Warn(fmt.Sprintf("ClientLogout: client %s not found", clientId))
		return constants.SYSTEM_ERROR, -1
	}

	if kafkaConfig.MessageMode == "channel" {
		ChatServer.SendClientToLogout(client)
	} else {
		KafkaChatServer.SendClientToLogout(client)
	}
	if err := client.Conn.Close(); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	close(client.SendTo)
	close(client.SendBack)
	close(client.HeartBeatDone)

	// log.Printf("ClientLogout退出啦 ：%s logout", clientId)
	return "退出成功", 0
}
