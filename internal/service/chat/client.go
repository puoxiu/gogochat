package chat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

type Client struct {
	Conn *websocket.Conn
	Id   string
	Send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	// 检查连接的Origin头
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 读取websocket消息并发送给send通道
func (c *Client) Read() {
	defer func() {
		if err := c.Conn.Close(); err != nil {
			zlog.Fatal(err.Error())
		}
		close(c.Send)
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			zlog.Error(err.Error())
		} else {
			c.Send <- message
		}
	}
}

// 从send通道读取消息发送给websocket
func (c *Client) Write() {
	defer func() {
		if err := c.Conn.Close(); err != nil {
			zlog.Fatal(err.Error())
		}
	}()
	for message := range c.Send {
		// 通过 WebSocket 发送消息
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			zlog.Error(err.Error())
		}
	}
}

// NewClientInit 当接受到前端有登录消息时，会调用该函数
func NewClientInit(c *gin.Context, clientId string) error {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Fatal(err.Error())
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			zlog.Fatal(err.Error())
		}
	}()

	client := &Client{
		Conn: conn,
		Id:   clientId,
		Send: make(chan []byte),
	}
	ChatServer.Login <- client
	go client.Read()
	go client.Write()
	return nil
}
